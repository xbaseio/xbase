//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	stdIo "io"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/utils/xbs"
	"github.com/xbaseio/xbase/utils/xbuffer/xelastic"
	"github.com/xbaseio/xbase/utils/xgfd"
	"github.com/xbaseio/xbase/utils/xio"
	"github.com/xbaseio/xbase/utils/xnetpoll"
	bsPool "github.com/xbaseio/xbase/utils/xpool/xbyteslice"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/utils/xsocket"
	"github.com/xbaseio/xbase/xerrors"
)

type conn struct {
	fd             int                     // 文件描述符
	xgfd           xgfd.GFD                // xnet 文件描述符
	ctx            any                     // 用户上下文
	remote         unix.Sockaddr           // 远程套接字地址
	proto          string                  // 协议名：tcp / udp / unix
	localAddr      net.Addr                // 本地地址
	remoteAddr     net.Addr                // 远程地址
	loop           *eventloop              // 所属事件循环
	outboundBuffer xelastic.XBuffer        // 出站缓冲区
	pollAttachment xnetpoll.PollAttachment // poller 附件
	inboundBuffer  xelastic.XRingBuffer    // 入站环形缓冲区
	buffer         []byte                  // 当前最新读入的数据
	cache          []byte                  // Peek 临时缓存
	isDatagram     bool                    // 是否为 UDP
	opened         bool                    // 是否已触发 open 事件
	isEOF          bool                    // 是否到达 EOF
}

func newStreamConn(proto string, fd int, el *eventloop, sa unix.Sockaddr, localAddr, remoteAddr net.Addr) (c *conn) {
	c = &conn{
		fd:             fd,
		proto:          proto,
		remote:         sa,
		loop:           el,
		localAddr:      localAddr,
		remoteAddr:     remoteAddr,
		pollAttachment: xnetpoll.PollAttachment{FD: fd},
	}
	c.pollAttachment.Callback = c.processIO
	c.outboundBuffer.Reset(el.engine.opts.WriteBufferCap)
	return
}

func newUDPConn(fd int, el *eventloop, localAddr net.Addr, sa unix.Sockaddr, connected bool) (c *conn) {
	c = &conn{
		fd:             fd,
		proto:          "udp",
		xgfd:           xgfd.NewGFD(fd, el.idx, 0, 0),
		remote:         sa,
		loop:           el,
		localAddr:      localAddr,
		remoteAddr:     xsocket.SockaddrToUDPAddr(sa),
		isDatagram:     true,
		pollAttachment: xnetpoll.PollAttachment{FD: fd, Callback: el.readUDP},
	}
	if connected {
		c.remote = nil
	}
	return
}

func (c *conn) release() {
	c.opened = false
	c.isEOF = false
	c.ctx = nil
	c.buffer = nil

	if addr, ok := c.localAddr.(*net.TCPAddr); ok && len(c.loop.listeners) == 0 && len(addr.Zone) > 0 {
		bsPool.Put(xbs.StringToBytes(addr.Zone))
	}
	if addr, ok := c.remoteAddr.(*net.TCPAddr); ok && len(addr.Zone) > 0 {
		bsPool.Put(xbs.StringToBytes(addr.Zone))
	}
	if addr, ok := c.localAddr.(*net.UDPAddr); ok && len(c.loop.listeners) == 0 && len(addr.Zone) > 0 {
		bsPool.Put(xbs.StringToBytes(addr.Zone))
	}
	if addr, ok := c.remoteAddr.(*net.UDPAddr); ok && len(addr.Zone) > 0 {
		bsPool.Put(xbs.StringToBytes(addr.Zone))
	}

	c.localAddr = nil
	c.remoteAddr = nil

	if !c.isDatagram {
		c.remote = nil
		c.inboundBuffer.Done()
		c.outboundBuffer.Release()
	}
}

func (c *conn) open(buf []byte) error {
	if c.isDatagram && c.remote == nil {
		return unix.Send(c.fd, buf, 0)
	}

	for len(buf) > 0 {
		n, err := unix.Write(c.fd, buf)
		if err != nil {
			if err == unix.EAGAIN {
				_, _ = c.outboundBuffer.Write(buf)
				break
			}
			return err
		}
		buf = buf[n:]
	}

	return nil
}

// write 将数据写入连接。
// ET 模式下会持续写到数据发完或遇到 EAGAIN；
// LT 模式下只尝试一次，其余交给后续可写事件。
func (c *conn) write(data []byte) (n int, err error) {
	isET := c.loop.engine.opts.EdgeTriggeredIO
	n = len(data)

	// 如果出站缓冲区中已有待发送数据，则直接追加，保证发送顺序。
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Write(data)
		return
	}

	defer func() {
		if err != nil {
			_ = c.loop.close(c, os.NewSyscallError("write", err))
		}
	}()

	for len(data) > 0 {
		sent, werr := unix.Write(c.fd, data)
		if werr != nil {
			// 临时不可写，将剩余数据写入出站缓冲区。
			if werr == unix.EAGAIN {
				_, err = c.outboundBuffer.Write(data)
				if !isET {
					err = c.loop.poller.ModReadWrite(&c.pollAttachment, false)
				}
				return
			}
			return 0, werr
		}

		data = data[sent:]

		// LT 模式只尝试一次。
		if !isET {
			break
		}
	}

	// 还有未写完的数据，缓存并监听可写事件。
	if len(data) > 0 {
		_, _ = c.outboundBuffer.Write(data)
		err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
	}

	return
}

// writev 批量写多个字节切片。
// ET 模式下会尽可能写空；LT 模式只尝试一次。
func (c *conn) writev(xbs [][]byte) (n int, err error) {
	isET := c.loop.engine.opts.EdgeTriggeredIO

	for _, b := range xbs {
		n += len(b)
	}

	// 如果出站缓冲区中已有待发送数据，则直接追加，保证发送顺序。
	if !c.outboundBuffer.IsEmpty() {
		_, _ = c.outboundBuffer.Writev(xbs)
		return
	}

	defer func() {
		if err != nil {
			_ = c.loop.close(c, os.NewSyscallError("writev", err))
		}
	}()

	remaining := n

	for remaining > 0 && len(xbs) > 0 {
		sent, werr := xio.Writev(c.fd, xbs)
		if werr != nil {
			// 临时不可写，将剩余数据写入出站缓冲区。
			if werr == unix.EAGAIN {
				_, err = c.outboundBuffer.Writev(xbs)
				if !isET {
					err = c.loop.poller.ModReadWrite(&c.pollAttachment, false)
				}
				return
			}
			return 0, werr
		}

		remaining -= sent
		xbs = consumeIovecs(xbs, sent)

		// LT 模式只尝试一次。
		if !isET {
			break
		}
	}

	// 还有未写完的数据，缓存并监听可写事件。
	if remaining > 0 && len(xbs) > 0 {
		_, _ = c.outboundBuffer.Writev(xbs)
		err = c.loop.poller.ModReadWrite(&c.pollAttachment, isET)
	}

	return
}

// consumeIovecs 按 sent 长度裁剪已写出的 iovec。
func consumeIovecs(xbs [][]byte, sent int) [][]byte {
	if sent <= 0 {
		return xbs
	}

	pos := 0
	for i := range xbs {
		bn := len(xbs[i])
		if sent < bn {
			xbs[i] = xbs[i][sent:]
			pos = i
			return xbs[pos:]
		}
		sent -= bn
		pos = i + 1
	}

	return xbs[pos:]
}

type asyncWriteHook struct {
	callback AsyncCallback
	data     []byte
}

func (c *conn) asyncWrite(a any) (err error) {
	hook := a.(*asyncWriteHook)
	defer func() {
		if hook.callback != nil {
			_ = hook.callback(c, err)
		}
	}()

	if !c.opened {
		return net.ErrClosed
	}

	_, err = c.write(hook.data)
	return
}

type asyncWritevHook struct {
	callback AsyncCallback
	data     [][]byte
}

func (c *conn) asyncWritev(a any) (err error) {
	hook := a.(*asyncWritevHook)
	defer func() {
		if hook.callback != nil {
			_ = hook.callback(c, err)
		}
	}()

	if !c.opened {
		return net.ErrClosed
	}

	_, err = c.writev(hook.data)
	return
}

func (c *conn) sendTo(buf []byte, addr unix.Sockaddr) (n int, err error) {
	defer func() {
		if err != nil {
			n = 0
		}
	}()

	if addr != nil {
		return len(buf), unix.Sendto(c.fd, buf, 0, addr)
	}
	if c.remote == nil { // client 侧 connected UDP
		return len(buf), unix.Send(c.fd, buf, 0)
	}
	return len(buf), unix.Sendto(c.fd, buf, 0, c.remote) // server 侧 unconnected UDP
}

func (c *conn) resetBuffer() {
	c.buffer = c.buffer[:0]
	c.inboundBuffer.Reset()
	c.inboundBuffer.Done()
}

func (c *conn) Read(p []byte) (n int, err error) {
	if c.inboundBuffer.IsEmpty() {
		n = copy(p, c.buffer)
		c.buffer = c.buffer[n:]
		if n == 0 && len(p) > 0 {
			err = stdIo.ErrShortBuffer
		}
		return
	}

	n, _ = c.inboundBuffer.Read(p)
	if n == len(p) {
		return
	}

	m := copy(p[n:], c.buffer)
	n += m
	c.buffer = c.buffer[m:]
	return
}

func (c *conn) Next(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n > totalLen {
		return nil, stdIo.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}

	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer[:n]
		c.buffer = c.buffer[n:]
		return
	}

	buf = bsPool.Get(n)
	_, err = c.Read(buf)
	return
}

func (c *conn) Peek(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n > totalLen {
		return nil, stdIo.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}

	if c.inboundBuffer.IsEmpty() {
		return c.buffer[:n], err
	}

	head, tail := c.inboundBuffer.Peek(n)
	if len(head) == n {
		return head, err
	}

	buf = bsPool.Get(n)[:0]
	buf = append(buf, head...)
	buf = append(buf, tail...)

	if inBufferLen >= n {
		return
	}

	remaining := n - inBufferLen
	buf = append(buf, c.buffer[:remaining]...)
	c.cache = buf
	return
}

func (c *conn) Discard(n int) (int, error) {
	if len(c.cache) > 0 {
		bsPool.Put(c.cache)
		c.cache = nil
	}

	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + len(c.buffer); n >= totalLen || n <= 0 {
		c.resetBuffer()
		return totalLen, nil
	}

	if c.inboundBuffer.IsEmpty() {
		c.buffer = c.buffer[n:]
		return n, nil
	}

	discarded, _ := c.inboundBuffer.Discard(n)
	if discarded < inBufferLen {
		return discarded, nil
	}

	remaining := n - inBufferLen
	c.buffer = c.buffer[remaining:]
	return n, nil
}

func (c *conn) Write(p []byte) (int, error) {
	if c.isDatagram {
		return c.sendTo(p, nil)
	}
	return c.write(p)
}

func (c *conn) SendTo(p []byte, addr net.Addr) (int, error) {
	if !c.isDatagram {
		return 0, xerrors.ErrUnsupportedOp
	}

	sa := xsocket.NetAddrToSockaddr(addr)
	if sa == nil {
		return 0, xerrors.ErrInvalidNetworkAddress
	}

	return c.sendTo(p, sa)
}

func (c *conn) Writev(xbs [][]byte) (int, error) {
	if c.isDatagram {
		return 0, xerrors.ErrUnsupportedOp
	}
	return c.writev(xbs)
}

func (c *conn) ReadFrom(r stdIo.Reader) (int64, error) {
	return c.outboundBuffer.ReadFrom(r)
}

func (c *conn) WriteTo(w stdIo.Writer) (n int64, err error) {
	if !c.inboundBuffer.IsEmpty() {
		if n, err = c.inboundBuffer.WriteTo(w); err != nil {
			return
		}
	}

	var m int
	m, err = w.Write(c.buffer)
	n += int64(m)
	c.buffer = c.buffer[m:]
	return
}

func (c *conn) Flush() error {
	return c.loop.write(c)
}

func (c *conn) InboundBuffered() int {
	return c.inboundBuffer.Buffered() + len(c.buffer)
}

func (c *conn) OutboundBuffered() int {
	return c.outboundBuffer.Buffered()
}

func (c *conn) Context() any         { return c.ctx }
func (c *conn) SetContext(ctx any)   { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr  { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr { return c.remoteAddr }

// Socket 接口实现

// func (c *conn) Gfd() xgfd.GFD { return c.xgfd }

func (c *conn) Fd() int                        { return c.fd }
func (c *conn) Dup() (fd int, err error)       { return xsocket.Dup(c.fd) }
func (c *conn) SetReadBuffer(bytes int) error  { return xsocket.SetRecvBuffer(c.fd, bytes) }
func (c *conn) SetWriteBuffer(bytes int) error { return xsocket.SetSendBuffer(c.fd, bytes) }
func (c *conn) SetLinger(sec int) error        { return xsocket.SetLinger(c.fd, sec) }

func (c *conn) SetNoDelay(noDelay bool) error {
	return xsocket.SetNoDelay(c.fd, func(b bool) int {
		if b {
			return 1
		}
		return 0
	}(noDelay))
}

func (c *conn) SetKeepAlivePeriod(d time.Duration) error {
	if c.proto != "tcp" {
		return xerrors.ErrUnsupportedOp
	}
	return xsocket.SetKeepAlivePeriod(c.fd, int(d.Seconds()))
}

func (c *conn) SetKeepAlive(enabled bool, idle, intvl time.Duration, cnt int) error {
	if c.proto != "tcp" {
		return xerrors.ErrUnsupportedOp
	}
	return xsocket.SetKeepAlive(c.fd, enabled, int(idle.Seconds()), int(intvl.Seconds()), cnt)
}

func (c *conn) AsyncWrite(buf []byte, callback AsyncCallback) error {
	if c.isDatagram {
		_, err := c.sendTo(buf, nil)

		// UDP 下这里实际上不是异步发送，因此 callback 不应被依赖来做关键逻辑。
		// 如果是 UDP，直接调用 Conn.Write 更合适。
		if callback != nil {
			_ = callback(nil, nil)
		}
		return err
	}

	return c.loop.poller.Trigger(
		xqueue.HighPriority,
		c.asyncWrite,
		&asyncWriteHook{callback, buf},
	)
}

func (c *conn) AsyncWritev(xbs [][]byte, callback AsyncCallback) error {
	if c.isDatagram {
		return xerrors.ErrUnsupportedOp
	}
	return c.loop.poller.Trigger(
		xqueue.HighPriority,
		c.asyncWritev,
		&asyncWritevHook{callback, xbs},
	)
}

func (c *conn) Wake(callback AsyncCallback) error {
	return c.loop.poller.Trigger(xqueue.LowPriority, func(_ any) (err error) {
		err = c.loop.wake(c)
		if callback != nil {
			_ = callback(c, err)
		}
		return
	}, nil)
}

func (c *conn) CloseWithCallback(callback AsyncCallback) error {
	return c.loop.poller.Trigger(xqueue.LowPriority, func(_ any) (err error) {
		err = c.loop.close(c, nil)
		if callback != nil {
			_ = callback(c, err)
		}
		return
	}, nil)
}

func (c *conn) Close() error {
	return c.loop.poller.Trigger(xqueue.LowPriority, func(_ any) (err error) {
		err = c.loop.close(c, nil)
		return
	}, nil)
}

func (c *conn) EventLoop() EventLoop {
	return c.loop
}

func (*conn) SetDeadline(_ time.Time) error {
	return xerrors.ErrUnsupportedOp
}

func (*conn) SetReadDeadline(_ time.Time) error {
	return xerrors.ErrUnsupportedOp
}

func (*conn) SetWriteDeadline(_ time.Time) error {
	return xerrors.ErrUnsupportedOp
}
