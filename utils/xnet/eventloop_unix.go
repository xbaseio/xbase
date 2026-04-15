//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	"context"
	"fmt"
	stdIo "io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xio"
	"github.com/xbaseio/xbase/utils/xnetpoll"
	"github.com/xbaseio/xbase/utils/xpool/xgoroutine"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/utils/xsocket"
	"github.com/xbaseio/xbase/xerrors"
	"golang.org/x/sys/unix"
)

// eventloop 表示一个 reactor/event-loop。
type eventloop struct {
	listeners    map[int]*listener // 监听器集合
	idx          int               // 在引擎中的 event-loop 索引
	engine       *engine           // 所属引擎
	poller       *xnetpoll.Poller  // epoll 或 kqueue
	buffer       []byte            // 读缓冲，容量由用户配置，默认 64KB
	connections  connMatrix        // 当前 loop 管理的连接集合
	eventHandler EventHandler      // 用户事件处理器
}

func (el *eventloop) Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error) {
	if el.engine.isShutdown() {
		return nil, xerrors.ErrEngineInShutdown
	}
	if addr == nil {
		return nil, xerrors.ErrInvalidNetworkAddress
	}
	return el.enroll(nil, addr, FromContext(ctx))
}

func (el *eventloop) Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error) {
	if el.engine.isShutdown() {
		return nil, xerrors.ErrEngineInShutdown
	}
	if c == nil {
		return nil, xerrors.ErrInvalidNetConn
	}
	return el.enroll(c, c.RemoteAddr(), FromContext(ctx))
}

func (el *eventloop) Execute(ctx context.Context, runnable Runnable) error {
	if el.engine.isShutdown() {
		return xerrors.ErrEngineInShutdown
	}
	if runnable == nil {
		return xerrors.ErrNilRunnable
	}
	return el.poller.Trigger(xqueue.LowPriority, func(any) error {
		return runnable.Run(ctx)
	}, nil)
}

func (el *eventloop) Schedule(context.Context, Runnable, time.Duration) error {
	return xerrors.ErrUnsupportedOp
}

func (el *eventloop) Close(c Conn) error {
	return el.close(c.(*conn), nil)
}

func (el *eventloop) countConn() int32 {
	return el.connections.loadCount()
}

func (el *eventloop) closeConns() {
	// 关闭当前 loop 中的所有连接。
	el.connections.iterate(func(c *conn) bool {
		_ = el.close(c, nil)
		return true
	})
}

type connWithCallback struct {
	c  *conn
	cb func()
}

func (el *eventloop) enroll(c net.Conn, addr net.Addr, ctx any) (resCh chan RegisteredResult, err error) {
	resCh = make(chan RegisteredResult, 1)

	err = xgoroutine.DefaultWorkerPool.Submit(func() {
		defer close(resCh)

		var err error
		if c == nil {
			if c, err = net.Dial(addr.Network(), addr.String()); err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
		}
		defer c.Close() //nolint:errcheck

		sc, ok := c.(syscall.Conn)
		if !ok {
			resCh <- RegisteredResult{
				Err: fmt.Errorf("failed to assert syscall.Conn from net.Conn: %s", addr.String()),
			}
			return
		}

		rc, err := sc.SyscallConn()
		if err != nil {
			resCh <- RegisteredResult{Err: err}
			return
		}

		var dupFD int
		err1 := rc.Control(func(fd uintptr) {
			dupFD, err = xsocket.Dup(int(fd))
		})
		if err != nil {
			resCh <- RegisteredResult{Err: err}
			return
		}
		if err1 != nil {
			resCh <- RegisteredResult{Err: err1}
			return
		}

		var (
			sockAddr unix.Sockaddr
			gc       *conn
		)

		switch c.(type) {
		case *net.UnixConn:
			sockAddr, _, _, err = xsocket.GetUnixSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			ua := c.LocalAddr().(*net.UnixAddr)
			ua.Name = c.RemoteAddr().String() + "." + strconv.Itoa(dupFD)
			gc = newStreamConn("unix", dupFD, el, sockAddr, c.LocalAddr(), c.RemoteAddr())

		case *net.TCPConn:
			sockAddr, _, _, _, err = xsocket.GetTCPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			gc = newStreamConn("tcp", dupFD, el, sockAddr, c.LocalAddr(), c.RemoteAddr())

		case *net.UDPConn:
			sockAddr, _, _, _, err = xsocket.GetUDPSockAddr(c.RemoteAddr().Network(), c.RemoteAddr().String())
			if err != nil {
				resCh <- RegisteredResult{Err: err}
				return
			}
			gc = newUDPConn(dupFD, el, c.LocalAddr(), sockAddr, true)

		default:
			resCh <- RegisteredResult{Err: fmt.Errorf("unknown type of conn: %T", c)}
			return
		}

		gc.ctx = ctx

		connOpened := make(chan struct{})
		ccb := &connWithCallback{
			c: gc,
			cb: func() {
				close(connOpened)
			},
		}

		if err := el.poller.Trigger(xqueue.LowPriority, el.register, ccb); err != nil {
			gc.Close() //nolint:errcheck
			resCh <- RegisteredResult{Err: err}
			return
		}

		<-connOpened
		resCh <- RegisteredResult{Conn: gc}
	})

	return
}

func (el *eventloop) register(a any) error {
	c, ok := a.(*conn)
	if !ok {
		ccb := a.(*connWithCallback)
		c = ccb.c
		defer ccb.cb()
	}
	return el.register0(c)
}

func (el *eventloop) register0(c *conn) error {
	addEvents := el.poller.AddRead
	if el.engine.opts.EdgeTriggeredIO {
		addEvents = el.poller.AddReadWrite
	}

	if err := addEvents(&c.pollAttachment, el.engine.opts.EdgeTriggeredIO); err != nil {
		_ = unix.Close(c.fd)
		c.release()
		return err
	}

	el.connections.addConn(c, el.idx)

	if c.isDatagram && c.remote != nil {
		return nil
	}

	return el.open(c)
}

func (el *eventloop) open(c *conn) error {
	c.opened = true

	out, action := el.eventHandler.OnOpen(c)
	if out != nil {
		if err := c.open(out); err != nil {
			return err
		}
	}

	if !c.outboundBuffer.IsEmpty() && !el.engine.opts.EdgeTriggeredIO {
		if err := el.poller.ModReadWrite(&c.pollAttachment, false); err != nil {
			return err
		}
	}

	return el.handleAction(c, action)
}

func (el *eventloop) read0(a any) error {
	return el.read(a.(*conn))
}

func (el *eventloop) read(c *conn) error {
	if !c.opened {
		return nil
	}

	var recv int
	isET := el.engine.opts.EdgeTriggeredIO
	chunk := el.engine.opts.EdgeTriggeredIOChunk

	for {
		n, err := unix.Read(c.fd, el.buffer)
		if err != nil || n == 0 {
			if err == unix.EAGAIN {
				return nil
			}
			if n == 0 {
				err = stdIo.EOF
			}
			return el.close(c, os.NewSyscallError("read", err))
		}

		recv += n
		c.buffer = el.buffer[:n]

		action := el.eventHandler.OnTraffic(c)
		switch action {
		case None:
		case Close:
			return el.close(c, nil)
		case Shutdown:
			return xerrors.ErrEngineShutdown
		}

		_, _ = c.inboundBuffer.Write(c.buffer)
		c.buffer = c.buffer[:0]

		if !(c.isEOF || (isET && recv < chunk)) {
			break
		}
	}

	// 为避免 ET 模式下单连接无限读导致其他事件饥饿，
	// 需要限制每次 event-loop 为单连接读取的最大字节数。
	// 当达到阈值且 socket 缓冲区中可能仍有未读数据时，手动补发一次读事件。
	if isET && recv >= chunk {
		return el.poller.Trigger(xqueue.LowPriority, el.read0, c)
	}

	return nil
}

func (el *eventloop) write0(a any) error {
	return el.write(a.(*conn))
}

// Linux 与大多数 BSD 系统上的 UIO_MAXIOV/IOV_MAX 默认值通常为 1024。
const iovMax = 1024

func (el *eventloop) write(c *conn) error {
	if c.outboundBuffer.IsEmpty() {
		return nil
	}

	isET := el.engine.opts.EdgeTriggeredIO
	chunk := el.engine.opts.EdgeTriggeredIOChunk

	var sent int

	for {
		iov, _ := c.outboundBuffer.Peek(-1)

		var (
			n   int
			err error
		)

		if len(iov) > 1 {
			if len(iov) > iovMax {
				iov = iov[:iovMax]
			}
			n, err = xio.Writev(c.fd, iov)
		} else {
			n, err = unix.Write(c.fd, iov[0])
		}

		_, _ = c.outboundBuffer.Discard(n)

		switch err {
		case nil:
		case unix.EAGAIN:
			return nil
		default:
			return el.close(c, os.NewSyscallError("write", err))
		}

		sent += n

		if !(isET && !c.outboundBuffer.IsEmpty() && sent < chunk) {
			break
		}
	}

	// LT 模式下，数据全部写完后不再需要监听可写事件。
	if !isET && c.outboundBuffer.IsEmpty() {
		return el.poller.ModRead(&c.pollAttachment, false)
	}

	// 为避免 ET 模式下单连接无限写导致其他事件饥饿，
	// 达到阈值后如果仍有待写数据，则手动补发一次写事件。
	if isET && !c.outboundBuffer.IsEmpty() {
		return el.poller.Trigger(xqueue.HighPriority, el.write0, c)
	}

	return nil
}

func (el *eventloop) close(c *conn, err error) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // 忽略过期连接
	}

	el.connections.delConn(c)
	action := el.eventHandler.OnClose(c, err)

	// 在真正关闭前，尽量把残留在出站缓冲区中的数据写回远端。
	for !c.outboundBuffer.IsEmpty() {
		iov, _ := c.outboundBuffer.Peek(0)
		if len(iov) > iovMax {
			iov = iov[:iovMax]
		}

		n, werr := xio.Writev(c.fd, iov)
		if werr != nil {
			break
		}
		_, _ = c.outboundBuffer.Discard(n)
	}

	c.release()

	var errStr strings.Builder

	err0, err1 := el.poller.Delete(c.fd), unix.Close(c.fd)
	if err0 != nil {
		err0 = fmt.Errorf(
			"failed to delete fd=%d from poller in event-loop(%d): %v",
			c.fd, el.idx, os.NewSyscallError("delete", err0),
		)
		errStr.WriteString(err0.Error())
		errStr.WriteString(" | ")
	}

	if err1 != nil {
		err1 = fmt.Errorf(
			"failed to close fd=%d in event-loop(%d): %v",
			c.fd, el.idx, os.NewSyscallError("close", err1),
		)
		errStr.WriteString(err1.Error())
	}

	if errStr.Len() > 0 {
		return xerrors.New(strings.TrimSuffix(errStr.String(), " | "))
	}

	return el.handleAction(c, action)
}

func (el *eventloop) wake(c *conn) error {
	if !c.opened || el.connections.getConn(c.fd) == nil {
		return nil // 忽略过期连接
	}

	action := el.eventHandler.OnTraffic(c)
	return el.handleAction(c, action)
}

func (el *eventloop) ticker(ctx context.Context) {
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

	for {
		delay, action = el.eventHandler.OnTick()

		switch action {
		case None, Close:
		case Shutdown:
			// 这里使用低优先级更合理，先让异步写等任务尽量收尾，再关闭服务。
			err := el.poller.Trigger(
				xqueue.LowPriority,
				func(_ any) error { return xerrors.ErrEngineShutdown },
				nil,
			)
			log.Debugf("failed to enqueue shutdown signal of high-priority for event-loop(%d): %v", el.idx, err)
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}

		select {
		case <-ctx.Done():
			log.Debugf("stopping ticker in event-loop(%d) from Engine, error:%v", el.idx, ctx.Err())
			return
		case <-timer.C:
		}
	}
}

func (el *eventloop) readUDP(fd int, _ xnetpoll.IOEvent, _ xnetpoll.IOFlags) error {
	n, sa, err := unix.Recvfrom(fd, el.buffer, 0)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return fmt.Errorf(
			"failed to read UDP packet from fd=%d in event-loop(%d), %v",
			fd, el.idx, os.NewSyscallError("recvfrom", err),
		)
	}

	var c *conn
	if ln, ok := el.listeners[fd]; ok {
		c = newUDPConn(fd, el, ln.addr, sa, false)
	} else {
		c = el.connections.getConn(fd)
	}

	c.buffer = el.buffer[:n]
	action := el.eventHandler.OnTraffic(c)

	if c.remote != nil {
		c.release()
	}

	if action == Shutdown {
		return xerrors.ErrEngineShutdown
	}

	return nil
}

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

/*
func (el *eventloop) execCmd(a any) (err error) {
	cmd := a.(*asyncCmd)
	c := el.connections.getConnByGFD(cmd.fd)
	if c == nil || c.gfd != cmd.fd {
		return xerrors.ErrInvalidConn
	}

	defer func() {
		if cmd.cb != nil {
			_ = cmd.cb(c, err)
		}
	}()

	switch cmd.typ {
	case asyncCmdClose:
		return el.close(c, nil)
	case asyncCmdWake:
		return el.wake(c)
	case asyncCmdWrite:
		_, err = c.Write(cmd.param.([]byte))
	case asyncCmdWritev:
		_, err = c.Writev(cmd.param.([][]byte))
	default:
		return xerrors.ErrUnsupportedOp
	}
	return
}
*/
