//go:build (darwin || dragonfly || freebsd || netbsd || openbsd) && !poll_opt
// +build darwin dragonfly freebsd netbsd openbsd
// +build !poll_opt

package xnetpoll

import (
	"os"
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/xerrors"
)

// Poller 表示一个轮询器，负责监听文件描述符（基于 kqueue）
type Poller struct {
	fd                          int   // kqueue fd
	pipe                        []int // 用于唤醒的 pipe（读写端）
	wakeupCall                  int32
	asyncTaskQueue              xqueue.AsyncTaskQueue // 低优先级任务队列
	urgentAsyncTaskQueue        xqueue.AsyncTaskQueue // 高优先级任务队列
	highPriorityEventsThreshold int32                 // 高优先级任务阈值
}

// OpenPoller 创建一个新的 poller
func OpenPoller() (poller *Poller, err error) {
	poller = new(Poller)

	// 创建 kqueue
	if poller.fd, err = unix.Kqueue(); err != nil {
		poller = nil
		err = os.NewSyscallError("kqueue", err)
		return
	}

	// 添加唤醒事件（pipe + kevent）
	if err = poller.addWakeupEvent(); err != nil {
		_ = poller.Close()
		poller = nil
		err = os.NewSyscallError("kevent | pipe2", err)
		return
	}

	// 初始化任务队列
	poller.asyncTaskQueue = xqueue.NewLockFreeQueue()
	poller.urgentAsyncTaskQueue = xqueue.NewLockFreeQueue()
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
// - 默认进入高优先级队列
// - 超过阈值后进入低优先级队列
//
// 注意：低优先级队列可能积压
func (p *Poller) Trigger(priority xqueue.EventPriority, fn xqueue.Func, param any) (err error) {
	task := xqueue.GetTask()
	task.Exec, task.Param = fn, param

	// 根据优先级分流队列
	if priority > xqueue.HighPriority && p.urgentAsyncTaskQueue.Length() >= p.highPriorityEventsThreshold {
		p.asyncTaskQueue.Enqueue(task)
	} else {
		// 极端情况下低优先级任务可能进入高优先级队列（可接受）
		p.urgentAsyncTaskQueue.Enqueue(task)
	}

	// 唤醒 poller（只触发一次）
	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
		err = p.wakePoller()
	}

	return os.NewSyscallError("kevent | write", err)
}

// Polling 启动事件循环（阻塞）
//
// 监听 fd，一旦发生 IO 事件则调用 callback
func (p *Poller) Polling(callback PollEventHandler) error {
	el := newEventList(InitPollEventsCap)

	var (
		ts       unix.Timespec
		tsp      *unix.Timespec
		doChores bool
	)

	for {
		n, err := unix.Kevent(p.fd, nil, el.events, tsp)

		// 无事件或被信号打断
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

			// pipe 唤醒事件（Ident == 0）
			if fd := int(ev.Ident); fd == 0 {
				doChores = true
				p.drainWakeupEvent()
			} else {
				err = callback(fd, ev.Filter, ev.Flags)
				if xerrors.Is(err, xerrors.ErrAcceptSocket) || xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
			}
		}

		// 执行任务队列
		if doChores {
			doChores = false

			// 高优先级任务
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				err = task.Exec(task.Param)
				if xerrors.Is(err, xerrors.ErrEngineShutdown) {
					return err
				}
				xqueue.PutTask(task)
			}

			// 低优先级任务（限量执行）
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

			// 如果还有任务，继续唤醒
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) &&
				atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {

				if err = p.wakePoller(); err != nil {
					doChores = true
				}
			}
		}

		// 动态扩容 / 缩容
		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
	}
}

// AddReadWrite 注册读写事件
func (p *Poller) AddReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var flags IOFlags = unix.EV_ADD
	if edgeTriggered {
		flags |= unix.EV_CLEAR
	}
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: keventIdent(pa.FD), Flags: flags, Filter: unix.EVFILT_READ},
		{Ident: keventIdent(pa.FD), Flags: flags, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddRead 注册读事件
func (p *Poller) AddRead(pa *PollAttachment, edgeTriggered bool) error {
	var flags IOFlags = unix.EV_ADD
	if edgeTriggered {
		flags |= unix.EV_CLEAR
	}
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: keventIdent(pa.FD), Flags: flags, Filter: unix.EVFILT_READ},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// AddWrite 注册写事件
func (p *Poller) AddWrite(pa *PollAttachment, edgeTriggered bool) error {
	var flags IOFlags = unix.EV_ADD
	if edgeTriggered {
		flags |= unix.EV_CLEAR
	}
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: keventIdent(pa.FD), Flags: flags, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// ModRead 修改为只读（删除写事件）
func (p *Poller) ModRead(pa *PollAttachment, _ bool) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: keventIdent(pa.FD), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent delete", err)
}

// ModReadWrite 修改为读写
func (p *Poller) ModReadWrite(pa *PollAttachment, edgeTriggered bool) error {
	var flags IOFlags = unix.EV_ADD
	if edgeTriggered {
		flags |= unix.EV_CLEAR
	}
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: keventIdent(pa.FD), Flags: flags, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return os.NewSyscallError("kevent add", err)
}

// Delete 删除 fd（kqueue 模式下通常无需显式删除）
func (*Poller) Delete(_ int) error {
	return nil
}
