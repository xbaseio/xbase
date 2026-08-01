package node

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type Creator func(actor *Actor, args ...any) Processor

const (
	unstart   int32 = iota // 未启动
	started                // 已启动
	destroyed              // 已销毁
)

type Actor struct {
	opts                *actorOptions                  // 配置项
	scheduler           *Scheduler                     // 调度器
	state               atomic.Int32                   // 状态
	routes              map[int32]RouteHandler         // 路由处理器
	events              map[cluster.Event]EventHandler // 事件处理器
	defaultRouteHandler RouteHandler                   // 默认路由处理器
	processor           Processor                      // 处理器
	rw                  sync.RWMutex                   // 锁
	mailbox             chan Context                   // 邮箱
	fnChan              chan func()                    // 调用函数
	binds               sync.Map                       // 绑定的用户
}

// ID 获取Actor的ID
func (a *Actor) ID() string {
	return a.opts.id
}

// PID 获取Actor的唯一识别ID
func (a *Actor) PID() string {
	return a.Kind() + "/" + a.ID()
}

// Kind 获取Actor类型
func (a *Actor) Kind() string {
	return a.opts.kind
}

// Spawn 衍生出一个Actor
func (a *Actor) Spawn(creator Creator, opts ...ActorOption) (*Actor, error) {
	return a.scheduler.spawn(creator, opts...)
}

// Proxy 获取代理API
func (a *Actor) Proxy() *Proxy {
	return a.scheduler.node.proxy
}

// Invoke 调用函数（Actor内线程安全）
func (a *Actor) Invoke(fn func()) {
	a.rw.RLock()
	defer a.rw.RUnlock()

	if a.state.Load() != started {
		return
	}

	// 使用非阻塞发送，避免持有 RLock 时因 fnChan 满而阻塞，
	// 导致 destroy() 等待 Lock 时形成死锁。
	select {
	case a.fnChan <- fn:
	default:
		xlog.Logger().Warn("actor fnChan full, drop invoke: kind= id", zap.Any("kind", a.Kind()), zap.Any("iD", a.ID()))
	}
}

// AfterFunc 延迟调用，与官方的time.AfterFunc用法一致
func (a *Actor) AfterFunc(d time.Duration, f func()) *Timer {
	if a.state.Load() != started {
		return nil
	}

	timer := time.AfterFunc(d, func() {
		a.rw.RLock()
		defer a.rw.RUnlock()

		if a.state.Load() != started {
			return
		}

		f()
	})

	return &Timer{timer: timer}
}

// AfterInvoke 延迟调用（线程安全）
func (a *Actor) AfterInvoke(d time.Duration, f func()) *Timer {
	if a.state.Load() != started {
		return nil
	}

	timer := time.AfterFunc(d, func() {
		a.rw.RLock()
		defer a.rw.RUnlock()

		if a.state.Load() != started {
			return
		}

		// 与 Invoke 保持一致：非阻塞发送，避免 timer goroutine
		// 持有 RLock 时因 fnChan 满而阻塞，导致 destroy 死锁。
		select {
		case a.fnChan <- f:
		default:
			xlog.Logger().Warn("actor fnChan full, drop after-invoke: kind= id", zap.Any("kind", a.Kind()), zap.Any("iD", a.ID()))
		}
	})

	return &Timer{timer: timer}
}

// SetDefaultRouteHandler 设置默认路由处理器
func (a *Actor) SetDefaultRouteHandler(handler RouteHandler) {
	a.rw.RLock()
	defer a.rw.RUnlock()

	switch a.state.Load() {
	case unstart:
		a.defaultRouteHandler = handler
	case started:
		a.fnChan <- func() {
			a.defaultRouteHandler = handler
		}
	default:
		// ignore
	}
}

// AddRouteHandler 添加路由处理器
func (a *Actor) AddRouteHandler(route int32, handler RouteHandler) {
	a.rw.RLock()
	defer a.rw.RUnlock()

	switch a.state.Load() {
	case unstart:
		a.routes[route] = handler
	case started:
		a.fnChan <- func() {
			a.routes[route] = handler

			if a.opts.dispatch {
				a.scheduler.routes.Store(route, a.Kind())
			}
		}
	default:
		// ignore
	}
}

// AddEventHandler 添加事件处理器
func (a *Actor) AddEventHandler(event cluster.Event, handler EventHandler) {
	a.rw.RLock()
	defer a.rw.RUnlock()

	switch a.state.Load() {
	case unstart:
		a.events[event] = handler
	case started:
		a.fnChan <- func() {
			a.events[event] = handler
		}
	default:
		// ignore
	}
}

// Next 投递消息到Actor中进行处理
func (a *Actor) Next(ctx Context) error {
	a.rw.RLock()
	defer a.rw.RUnlock()

	if a.state.Load() != started {
		return xerrors.ErrNotFoundActor
	}

	ctx.storeActor(a)

	ctx.incrVersion()

	ctx.Cancel()

	timeout := a.scheduler.node.opts.mailboxTimeout
	if timeout <= 0 {
		a.mailbox <- ctx
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case a.mailbox <- ctx:
		return nil
	case <-timer.C:
		xlog.Logger().Warn("actor mailbox full, drop message: uid: kind: id", zap.Any("messageID", ctx.MessageID()), zap.Any("uID", ctx.UID()), zap.Any("kind", a.Kind()), zap.Any("iD", a.ID()))
		return xerrors.ErrMailboxFull
	}
}

// Deliver 投递消息到当前Actor中进行处理
func (a *Actor) Deliver(uid int64, message *cluster.Message) error {
	buf, err := a.scheduler.node.proxy.PackBuffer(message.Data)
	if err != nil {
		return err
	}

	req := a.scheduler.node.reqPool.Get().(*request)
	req.nid = a.scheduler.node.opts.id
	req.uid = uid
	req.message.Seq = message.Seq
	req.message.GameID = message.GameID
	req.message.MessageID = message.MessageID
	req.message.Data = buf

	return a.Next(req)
}

// Push 推送消息到本地Node队列上进行处理
func (a *Actor) Push(uid int64, message *cluster.Message) error {
	buf, err := a.scheduler.node.proxy.PackBuffer(message.Data)
	if err != nil {
		return err
	}

	return a.scheduler.node.router.deliver("", a.scheduler.node.opts.id, a.PID(), 0, uid, message.Seq, message.GameID, message.MessageID, nil, buf)
}

// Destroy 销毁Actor
func (a *Actor) Destroy() (ok bool) {
	if ok = a.destroy(); !ok {
		return
	}

	_, ok = a.scheduler.remove(a.Kind(), a.ID())
	return
}

// 销毁Actor
func (a *Actor) destroy() bool {
	if !a.state.CompareAndSwap(started, destroyed) {
		return false
	}

	a.processor.Destroy()

	a.scheduler.batchUnbindActor(func(relations map[int64]map[string]*Actor) {
		a.binds.Range(func(uid, _ any) bool {
			delete(relations[uid.(int64)], a.Kind())
			return true
		})
	})

	a.rw.Lock()
	defer a.rw.Unlock()

	close(a.mailbox)

	close(a.fnChan)

	clear(a.routes)

	clear(a.events)

	a.processor = nil

	a.defaultRouteHandler = nil

	return true
}

// 绑定用户
func (a *Actor) bindUser(uid int64) {
	a.binds.Store(uid, struct{}{})
}

// 解绑用户
func (a *Actor) unbindUser(uid int64) bool {
	_, ok := a.binds.LoadAndDelete(uid)
	return ok
}

// 分发
func (a *Actor) dispatch() {
	for {
		select {
		case ctx, ok := <-a.mailbox:
			if !ok {
				return
			}

			version := ctx.loadVersion()

			if ctx.Kind() == Event {
				if handler, ok := a.events[ctx.Event()]; ok {
					xcall.Call(func() { handler(ctx) })

					ctx.compareVersionExecDefer(version)
				}
			} else {
				if handler, ok := a.routes[ctx.MessageID()]; ok {
					xcall.Call(func() { handler(ctx) })

					ctx.compareVersionExecDefer(version)
				} else if a.defaultRouteHandler != nil {
					xcall.Call(func() { a.defaultRouteHandler(ctx) })

					ctx.compareVersionExecDefer(version)
				}
			}

			ctx.compareVersionRecycle(version)
		case handle, ok := <-a.fnChan:
			if !ok {
				return
			}

			xcall.Call(handle)
		}
	}
}
