//go:build linux && !poll_opt
// +build linux,!poll_opt

package xnetpoll

import (
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/xbaseio/xbase/log"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/xerrors"
)

// Poller 表示一个轮询器，负责监听文件描述符（fd）
type Poller struct {
	fd                          int    // epoll 文件描述符
	efd                         int    // eventfd，用于唤醒 epoll
	efdBuf                      []byte // eventfd 读取缓冲区（8字节）
	wakeupCall                  int32
	asyncTaskQueue              xqueue.AsyncTaskQueue // 低优先级任务队列
	urgentAsyncTaskQueue        xqueue.AsyncTaskQueue // 高优先级任务队列
	highPriorityEventsThreshold int32                 // 高优先级任务阈值
}

// OpenPoller 创建一个新的 poller
func OpenPoller() (poller *Poller, err error) {
	poller = new(Poller)

	// 创建 epoll 实例
	if poller.fd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC); err != nil {
		poller = nil
		err = os.NewSyscallError("epoll_create1", err)
		return
	}

	// 创建 eventfd，用于跨线程唤醒
	if poller.efd, err = unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("eventfd", err)
		return
	}

	// 初始化 eventfd 缓冲区
	poller.efdBuf = make([]byte, 8)

	// 将 eventfd 注册到 epoll（监听读事件）
	if err = poller.AddRead(&PollAttachment{FD: poller.efd}, true); err != nil {
		_ = poller.Close()
		poller = nil
		return
	}

	// 初始化任务队列（无锁队列）
	poller.asyncTaskQueue = xqueue.NewLockFreeQueue()
	poller.urgentAsyncTaskQueue = xqueue.NewLockFreeQueue()

	// 设置高优先级阈值
	poller.highPriorityEventsThreshold = MaxPollEventsCap
	return
}

// Close 关闭 poller
func (p *Poller) Close() error {
	_ = unix.Close(p.efd)
	return os.NewSyscallError("close", unix.Close(p.fd))
}

// 为不同架构下的 Linux 保证字节序一致（eventfd 写入用）
// 参考：http://man7.org/linux/man-pages/man2/eventfd.2.html
var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

// Trigger 向队列中投递任务，并唤醒 poller
//
// 默认情况下：
// - 高优先级任务 → urgentAsyncTaskQueue
// - 当高优先级队列达到阈值后 → 普通任务进入 asyncTaskQueue
//
// 注意：
// asyncTaskQueue 是低优先级队列，可能会堆积
func (p *Poller) Trigger(priority xqueue.EventPriority, fn xqueue.Func, param any) (err error) {
	task := xqueue.GetTask()
	task.Exec, task.Param = fn, param

	// 根据优先级和阈值决定入哪个队列
	if priority > xqueue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// 极端情况下低优先级任务可能进入高优先级队列，但可接受
		p.urgentAsyncTaskQueue.Enqueue(task)
	}

	// 唤醒 poller（只触发一次）
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		for {
			_, err = unix.Write(p.efd, b)
			if err == unix.EAGAIN {
				// eventfd 满了，先读再写
				_, _ = unix.Read(p.efd, p.efdBuf)
				continue
			}
			break
		}
	}
	return os.NewSyscallError("write", err)
}

// Polling 开始轮询（阻塞当前 goroutine）
//
// 监听所有注册的 fd，一旦发生 IO 事件，调用 callback
func (p *Poller) Polling(callback PollEventHandler) error {
	el := newEventList(InitPollEventsCap)
	var doChores bool

	msec := -1

	for {
		n, err := unix.EpollWait(p.fd, el.events, msec)

		// 没事件或被信号打断
		if n == 0 || (n < 0 && err == unix.EINTR) {
			msec = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			log.Errorf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
			return err
		}

		msec = 0

		// 处理事件
		for i := 0; i < n; i++ {
			ev := &el.events[i]
			if fd := int(ev.Fd); fd == p.efd {
				// eventfd 触发 → 执行任务队列
				doChores = true
			} else {
				err = callback(fd, ev.Events, 0)
				if xerrors.Is(err, xerrors.ErrAcceptSocket) || xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
			}
		}

		// 执行任务
		if doChores {
			doChores = false

			// 先执行高优先级任务
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				err = task.Exec(task.Param)
				if xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
				xqueue.PutTask(task)
			}

			// 再执行低优先级任务（限量）
			for i := 0; i < MaxAsyncTasksAtOneTime; i++ {
				if task = p.asyncTaskQueue.Dequeue(); task == nil {
					break
				}
				err = task.Exec(task.Param)
				if xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
				xqueue.PutTask(task)
			}

			atomic.StoreInt32(&p.wakeupCall, 0)

			// 如果还有任务，继续唤醒下一轮
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) &&
				atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {

				for {
					_, err = unix.Write(p.efd, b)
					if err == unix.EAGAIN {
						_, _ = unix.Read(p.efd, p.efdBuf)
						continue
					}
					if err != nil {
						log.Errorf("failed to notify next round of event-loop for leftover tasks, %v",
							os.NewSyscallError("write", err))
					}
					break
				}
			}
		}

		// 动态扩容/缩容事件列表
		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
	}
}

// AddReadWrite 注册读写事件
func (p *Poller) AddReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadWriteEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// AddRead 注册读事件
func (p *Poller) AddRead(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// AddWrite 注册写事件
func (p *Poller) AddWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = WriteEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl add",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// ModRead 修改为读事件
func (p *Poller) ModRead(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// ModReadWrite 修改为读写事件
func (p *Poller) ModReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev uint32 = ReadWriteEvents
	if edgeTriggered {
		ev |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	return os.NewSyscallError("epoll_ctl mod",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &unix.EpollEvent{Fd: int32(pa.FD), Events: ev}))
}

// Delete 删除 fd
func (p *Poller) Delete(fd int) error {
	return os.NewSyscallError("epoll_ctl del",
		unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil))
}
