//go:build linux && poll_opt
// +build linux,poll_opt

package xnetpoll

import (
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/xerrors"
)

// Poller 表示一个轮询器，负责监听文件描述符（fd）
type Poller struct {
	fd                          int             // epoll 文件描述符
	epa                         *PollAttachment // 用于唤醒的 PollAttachment（eventfd 封装）
	efdBuf                      []byte          // eventfd 读取缓冲区（8字节）
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

	// 创建 eventfd，用于唤醒 epoll
	var efd int
	if efd, err = unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("eventfd", err)
		return
	}

	// 初始化缓冲区
	poller.efdBuf = make([]byte, 8)

	// 封装为 PollAttachment（用于 epoll data 传递）
	poller.epa = &PollAttachment{FD: efd}

	// 注册 eventfd 到 epoll
	if err = poller.AddRead(poller.epa, true); err != nil {
		_ = poller.Close()
		poller = nil
		return
	}

	// 初始化无锁队列
	poller.asyncTaskQueue = xqueue.NewLockFreeQueue()
	poller.urgentAsyncTaskQueue = xqueue.NewLockFreeQueue()

	// 设置高优先级阈值
	poller.highPriorityEventsThreshold = MaxPollEventsCap
	return
}

// Close 关闭 poller
func (p *Poller) Close() error {
	_ = unix.Close(p.epa.FD)
	return os.NewSyscallError("close", unix.Close(p.fd))
}

// 为不同架构下的 Linux 保证字节序一致（eventfd 写入用）
// 参考：http://man7.org/linux/man-pages/man2/eventfd.2.html
var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

// Trigger 投递任务并唤醒 poller
//
// 规则：
// - 优先进入高优先级队列
// - 超过阈值后进入低优先级队列
func (p *Poller) Trigger(priority xqueue.EventPriority, fn xqueue.Func, param any) (err error) {
	task := xqueue.GetTask()
	task.Exec, task.Param = fn, param

	// 根据优先级决定队列
	if priority > xqueue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// 极端情况下低优先级任务可能进入高优先级队列（可接受）
		p.urgentAsyncTaskQueue.Enqueue(task)
	}

	// 唤醒 poller（避免重复唤醒）
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		for {
			_, err = unix.Write(p.epa.FD, b)
			if err == unix.EAGAIN {
				// eventfd 写满，先读再写
				_, _ = unix.Read(p.epa.FD, p.efdBuf)
				continue
			}
			break
		}
	}
	return os.NewSyscallError("write", err)
}

// Polling 启动事件循环（阻塞）
//
// 监听所有 fd，一旦有事件就回调对应 handler
func (p *Poller) Polling() error {
	el := newEventList(InitPollEventsCap)
	var doChores bool

	msec := -1

	for {
		n, err := epollWait(p.fd, el.events, msec)

		// 无事件或被信号中断
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

			// 从 epoll data 恢复 attachment
			pollAttachment := restorePollAttachment(unsafe.Pointer(&ev.data))

			if pollAttachment.FD == p.epa.FD {
				// eventfd 触发 → 执行任务
				doChores = true
			} else {
				err = pollAttachment.Callback(pollAttachment.FD, ev.events, 0)
				if stdErrors.Is(err, xerrors.ErrAcceptSocket) || stdErrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
			}
		}

		// 执行任务队列
		if doChores {
			doChores = false

			// 高优先级
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				err = task.Exec(task.Param)
				if stdErrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
				xqueue.PutTask(task)
			}

			// 低优先级（限量执行）
			for i := 0; i < MaxAsyncTasksAtOneTime; i++ {
				if task = p.asyncTaskQueue.Dequeue(); task == nil {
					break
				}
				err = task.Exec(task.Param)
				if stdErrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
				xqueue.PutTask(task)
			}

			atomic.StoreInt32(&p.wakeupCall, 0)

			// 若还有任务 → 再次唤醒
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) &&
				atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {

				for {
					_, err = unix.Write(p.epa.FD, b)
					if err == unix.EAGAIN {
						_, _ = unix.Read(p.epa.FD, p.efdBuf)
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

		// 动态扩容/缩容
		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
	}
}

// AddReadWrite 注册读写事件
func (p *Poller) AddReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev epollevent
	ev.events = ReadWriteEvents
	if edgeTriggered {
		ev.events |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	convertPollAttachment(unsafe.Pointer(&ev.data), pa)
	return os.NewSyscallError("epoll_ctl add",
		epollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &ev))
}

// AddRead 注册读事件
func (p *Poller) AddRead(pa *PollAttachment, edgeTriggered bool) error {
	var ev epollevent
	ev.events = ReadEvents
	if edgeTriggered {
		ev.events |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	convertPollAttachment(unsafe.Pointer(&ev.data), pa)
	return os.NewSyscallError("epoll_ctl add",
		epollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &ev))
}

// AddWrite 注册写事件
func (p *Poller) AddWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev epollevent
	ev.events = WriteEvents
	if edgeTriggered {
		ev.events |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	convertPollAttachment(unsafe.Pointer(&ev.data), pa)
	return os.NewSyscallError("epoll_ctl add",
		epollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &ev))
}

// ModRead 修改为读事件
func (p *Poller) ModRead(pa *PollAttachment, edgeTriggered bool) error {
	var ev epollevent
	ev.events = ReadEvents
	if edgeTriggered {
		ev.events |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	convertPollAttachment(unsafe.Pointer(&ev.data), pa)
	return os.NewSyscallError("epoll_ctl mod",
		epollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &ev))
}

// ModReadWrite 修改为读写事件
func (p *Poller) ModReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var ev epollevent
	ev.events = ReadWriteEvents
	if edgeTriggered {
		ev.events |= unix.EPOLLET | unix.EPOLLRDHUP
	}
	convertPollAttachment(unsafe.Pointer(&ev.data), pa)
	return os.NewSyscallError("epoll_ctl mod",
		epollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &ev))
}

// Delete 删除 fd
func (p *Poller) Delete(fd int) error {
	return os.NewSyscallError("epoll_ctl del",
		epollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil))
}
