package xnet

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xpool/xgoroutine"
	"github.com/xbaseio/xbase/xerrors"
)

//
// =========================
// 结构定义
// =========================
//

// eventloop = 单线程 reactor（基于 channel 驱动）
type eventloop struct {
	ch           chan any           // 事件队列（核心）
	idx          int                // 当前 loop index
	eng          *engine            // 所属 engine
	connCount    int32              // 当前连接数（原子）
	connections  map[*conn]struct{} // 活跃连接集合
	eventHandler EventHandler       // 用户回调
}

//
// =========================
// 对外 API
// =========================
//

// Register 注册一个地址（内部会 Dial）
func (el *eventloop) Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error) {
	if el.eng.isShutdown() {
		return nil, xerrors.ErrEngineInShutdown
	}
	if addr == nil {
		return nil, xerrors.ErrInvalidNetworkAddress
	}
	return el.enroll(nil, addr, FromContext(ctx))
}

// Enroll 把已有连接交给 eventloop
func (el *eventloop) Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error) {
	if el.eng.isShutdown() {
		return nil, xerrors.ErrEngineInShutdown
	}
	if c == nil {
		return nil, xerrors.ErrInvalidNetConn
	}
	return el.enroll(c, c.RemoteAddr(), FromContext(ctx))
}

// Execute 在 loop 内执行任务（非阻塞）
func (el *eventloop) Execute(ctx context.Context, runnable Runnable) error {
	if el.eng.isShutdown() {
		return xerrors.ErrEngineInShutdown
	}
	if runnable == nil {
		return xerrors.ErrNilRunnable
	}

	return xgoroutine.DefaultWorkerPool.Submit(func() {
		el.ch <- func() error {
			return runnable.Run(ctx)
		}
	})
}

// Schedule（未实现）
func (el *eventloop) Schedule(context.Context, Runnable, time.Duration) error {
	return xerrors.ErrUnsupportedOp
}

// Close 关闭连接
func (el *eventloop) Close(c Conn) error {
	return el.close(c.(*conn), nil)
}

//
// =========================
// 连接注册核心
// =========================
//

func (el *eventloop) enroll(c net.Conn, addr net.Addr, ctx any) (resCh chan RegisteredResult, err error) {
	resCh = make(chan RegisteredResult, 1)

	err = xgoroutine.DefaultWorkerPool.Submit(func() {
		defer close(resCh)

		// 没传 conn → 自动 Dial
		if c == nil {
			var err error
			if c, err = net.Dial(addr.Network(), addr.String()); err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
		}

		connOpened := make(chan struct{})
		var gc *conn

		switch addr.Network() {

		// TCP / Unix
		case "tcp", "tcp4", "tcp6", "unix":
			gc = newStreamConn(el, c, ctx)

			el.ch <- &openConn{
				c:  gc,
				cb: func() { close(connOpened) },
			}

			// 启动读协程
			xgoroutine.DefaultWorkerPool.Submit(func() {
				var buffer [0x10000]byte

				for {
					n, err := c.Read(buffer[:])
					if err != nil {
						el.ch <- &netErr{gc, err}
						return
					}
					el.ch <- packTCPConn(gc, buffer[:n])
				}
			})

		// UDP
		case "udp", "udp4", "udp6":
			gc = newUDPConn(el, nil, c, c.LocalAddr(), c.RemoteAddr(), ctx)

			el.ch <- &openConn{
				c:  gc,
				cb: func() { close(connOpened) },
			}

			xgoroutine.DefaultWorkerPool.Submit(func() {
				var buffer [0x10000]byte

				for {
					n, err := c.Read(buffer[:])
					if err != nil {
						el.ch <- &netErr{gc, err}
						return
					}

					// UDP 每包一个 conn（设计就是这样）
					gc := newUDPConn(el, nil, c, c.LocalAddr(), c.RemoteAddr(), ctx)
					el.ch <- packUDPConn(gc, buffer[:n])
				}
			})
		}

		<-connOpened
		resCh <- RegisteredResult{Conn: gc}
	})

	return
}

//
// =========================
// 连接计数
// =========================
//

func (el *eventloop) incConn(delta int32) {
	atomic.AddInt32(&el.connCount, delta)
}

func (el *eventloop) countConn() int32 {
	return atomic.LoadInt32(&el.connCount)
}

//
// =========================
// 主循环（核心）
// =========================
//

func (el *eventloop) run() (err error) {
	defer func() {
		// 触发 engine shutdown
		el.eng.shutdown(err)

		// 清理连接
		for c := range el.connections {
			_ = el.close(c, nil)
		}
	}()

	if el.eng.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	for msg := range el.ch {

		switch v := msg.(type) {

		case error:
			err = v

		case *netErr:
			err = el.close(v.c, v.err)

		case *openConn:
			err = el.open(v)

		case *tcpConn:
			err = el.read(unpackTCPConn(v))

		case *udpConn:
			err = el.readUDP(v.c)

		case func() error:
			err = v()
		}

		if xerrors.Is(err, xerrors.ErrEngineShutdown) {
			log.Debugf("event-loop(%d) exiting: %v", el.idx, err)
			break
		}

		if err != nil {
			log.Debugf("event-loop(%d) error: %v", el.idx, err)
		}
	}

	return nil
}

//
// =========================
// 生命周期
// =========================
//

// open 连接建立
func (el *eventloop) open(oc *openConn) error {
	if oc.cb != nil {
		defer oc.cb()
	}

	c := oc.c
	el.connections[c] = struct{}{}
	el.incConn(1)

	out, action := el.eventHandler.OnOpen(c)

	if out != nil {
		if _, err := c.rawConn.Write(out); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

// read TCP 数据
func (el *eventloop) read(c *conn) error {
	if _, ok := el.connections[c]; !ok {
		return nil
	}

	action := el.eventHandler.OnTraffic(c)

	switch action {
	case None:
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return xerrors.ErrEngineShutdown
	}

	_, _ = c.inboundBuffer.Write(c.buffer.B)
	c.buffer.Reset()

	return nil
}

// readUDP UDP 数据
func (el *eventloop) readUDP(c *conn) error {
	action := el.eventHandler.OnTraffic(c)

	if action == Shutdown {
		return xerrors.ErrEngineShutdown
	}

	c.release()
	return nil
}

// wake 外部唤醒
func (el *eventloop) wake(c *conn) error {
	if _, ok := el.connections[c]; !ok {
		return nil
	}
	action := el.eventHandler.OnTraffic(c)
	return el.handleAction(c, action)
}

// close 关闭连接
func (el *eventloop) close(c *conn, err error) error {
	if _, ok := el.connections[c]; c.rawConn == nil || !ok {
		return nil
	}

	delete(el.connections, c)
	el.incConn(-1)

	action := el.eventHandler.OnClose(c, err)

	err = c.rawConn.Close()
	c.release()

	if err != nil {
		return fmt.Errorf("close connection=%s failed: %v", c.remoteAddr, err)
	}

	return el.handleAction(c, action)
}

//
// =========================
// ticker
// =========================
//

func (el *eventloop) ticker(ctx context.Context) {
	if el == nil {
		return
	}

	var (
		action Action
		delay  time.Duration
		timer  *time.Timer
	)

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	var shutdown bool

	for {
		delay, action = el.eventHandler.OnTick()

		switch action {
		case None, Close:
		case Shutdown:
			if !shutdown {
				shutdown = true
				el.ch <- xerrors.ErrEngineShutdown
				log.Debugf("ticker stop loop(%d)", el.idx)
			}
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}

		select {
		case <-ctx.Done():
			log.Debugf("ticker stop by ctx loop(%d)", el.idx)
			return
		case <-timer.C:
		}
	}
}

//
// =========================
// Action处理
// =========================
//

func (el *eventloop) handleAction(c *conn, action Action) error {
	switch action {
	case None:
		return nil
	case Close:
		return el.close(c, nil)
	case Shutdown:
		return xerrors.ErrEngineShutdown
	default:
		return nil
	}
}
