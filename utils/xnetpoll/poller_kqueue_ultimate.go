//go:build (darwin || dragonfly || freebsd || netbsd || openbsd) && poll_opt
// +build darwin dragonfly freebsd netbsd openbsd
// +build poll_opt

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

// Poller 表示一个轮询器（基于 kqueue + poll_opt 优化）
type Poller struct {
	fd                          int   // kqueue fd
	pipe                        []int // 唤醒用 pipe（读写端）
	wakeupCall                  int32
	asyncTaskQueue              xqueue.AsyncTaskQueue // 低优先级任务队列
	urgentAsyncTaskQueue        xqueue.AsyncTaskQueue // 高优先级任务队列
	highPriorityEventsThreshold int32                 // 高优先级任务阈值
}

// OpenPoller 创建 poller
func OpenPoller() (poller *Poller, err error) {
	poller = new(Poller)

	// 创建 kqueue
	if poller.fd, err = unix.Kqueue(); err != nil {
		poller = nil
		err = os.NewSyscallError("kqueue", err)
		return
	}

	// 添加唤醒机制（pipe + kevent）
	if err = poller.addWakeupEvent(); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("kevent | pipe2", err)
		return
	}

	// 初始化无锁队列
	poller.asyncTaskQueue = xqueue.NewLockFreeQueue()
	poller.urgentAsyncTaskQueue = xqueue.NewLockFreeQueue()

	// 设置阈值
	poller.highPriorityEventsThreshold = MaxPollEventsCap
	return
}

// Close 关闭 poller
func (p *Poller) Close() error {
	if len(p.pipe) == 2 {
		_ = unix.Close(p.pipe[0])
		_ = unix.Close(p.pipe[1])
	}
	return os.NewSyscallError("close", unix.Close(p.fd))
}

// Trigger 投递任务并唤醒 poller
//
// 规则：
// - 优先进入高优先级队列
// - 阈值后进入低优先级队列
func (p *Poller) Trigger(priority xqueue.EventPriority, fn xqueue.Func, param any) (err error) {
	task := xqueue.GetTask()
	task.Exec, task.Param = fn, param

	// 队列分流
	if priority > xqueue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// 极端情况下低优先级任务可能进入高优先级队列（允许）
		p.urgentAsyncTaskQueue.Enqueue(task)
	}

	// 唤醒 poller
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		err = p.wakePoller()
	}

	return os.NewSyscallError("kevent | write", err)
}

// Polling 事件循环（阻塞）
//
// 基于 kqueue，使用 Udata 指针直接回调（无查找）
func (p *Poller) Polling() error {
	el := newEventList(InitPollEventsCap)

	var (
		ts       unix.Timespec
		tsp      *unix.Timespec
		doChores bool
	)

	for {
		n, err := unix.Kevent(p.fd, nil, el.events, tsp)

		// 无事件或被中断
		if n == 0 || (n < 0 && err == unix.EINTR) {
			tsp = nil
			runtime.Gosched()
			continue
		} else if err != nil {
			log.Errorf("error occurs in kqueue: %v", os.NewSyscallError("kevent wait", err))
			return err
		}

		// 下次非阻塞
		tsp = &ts

		for i := 0; i < n; i++ {
			ev := &el.events[i]

			// 唤醒事件（pipe）
			if ev.Ident == 0 {
				doChores = true
				p.drainWakeupEvent()
			} else {
				// 从 Udata 直接恢复 attachment（零查找）
				pollAttachment := restorePollAttachment(unsafe.Pointer(&ev.Udata))

				err = pollAttachment.Callback(int(ev.Ident), ev.Filter, ev.Flags)
				if xerrors.Is(err, xerrors.ErrAcceptSocket) || xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
			}
		}

		// 执行任务
		if doChores {
			doChores = false

			// 高优先级
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				err = task.Exec(task.Param)
				if xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
				xqueue.PutTask(task)
			}

			// 低优先级（限量）
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

			// 若还有任务，继续唤醒
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) &&
				atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {

				if err = p.wakePoller(); err != nil {
					doChores = true
				}
			}
		}

		// 动态扩缩容
		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
	}
}

// AddReadWrite 注册读写事件（带 Udata）
func (p *Poller) AddReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var evs [2]unix.Kevent_t

	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Flags = unix.EV_ADD
	if edgeTriggered {
		evs[0].Flags |= unix.EV_CLEAR
	}
	convertPollAttachment(unsafe.Pointer(&evs[0].Udata), pa)

	evs[1] = evs[0]
	evs[1].Filter = unix.EVFILT_WRITE

	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddRead 注册读事件
func (p *Poller) AddRead(pa *PollAttachment, edgeTriggered bool) error {
	var evs [1]unix.Kevent_t

	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Flags = unix.EV_ADD
	if edgeTriggered {
		evs[0].Flags |= unix.EV_CLEAR
	}

	convertPollAttachment(unsafe.Pointer(&evs[0].Udata), pa)

	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddWrite 注册写事件
func (p *Poller) AddWrite(pa *PollAttachment, edgeTriggered bool) error {
	var evs [1]unix.Kevent_t

	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Flags = unix.EV_ADD
	if edgeTriggered {
		evs[0].Flags |= unix.EV_CLEAR
	}

	convertPollAttachment(unsafe.Pointer(&evs[0].Udata), pa)

	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// ModRead 修改为只读（删除写事件）
func (p *Poller) ModRead(pa *PollAttachment, _ bool) error {
	var evs [1]unix.Kevent_t

	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Flags = unix.EV_DELETE

	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent delete", err)
}

// ModReadWrite 修改为读写
func (p *Poller) ModReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var evs [1]unix.Kevent_t

	evs[0].Ident = keventIdent(pa.FD)
	evs[0].Filter = unix.EVFILT_WRITE
	evs[0].Flags = unix.EV_ADD
	if edgeTriggered {
		evs[0].Flags |= unix.EV_CLEAR
	}

	convertPollAttachment(unsafe.Pointer(&evs[0].Udata), pa)

	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// Delete 删除 fd（kqueue 通常不需要显式删除）
func (p *Poller) Delete(_ int) error {
	return nil
}
