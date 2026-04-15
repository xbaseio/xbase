package xnet

import (
	"io"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"github.com/xbaseio/xbase/utils/xbuffer/xelastic"
	"github.com/xbaseio/xbase/xerrors"

	bbPool "github.com/xbaseio/xbase/utils/xpool/xbytebuffer"
	bsPool "github.com/xbaseio/xbase/utils/xpool/xbyteslice"
	"github.com/xbaseio/xbase/utils/xpool/xgoroutine"
)

// netErr 表示底层网络错误。
type netErr struct {
	c   *conn
	err error
}

// tcpConn 用于打包 TCP 连接及其临时缓冲。
type tcpConn struct {
	c *conn
	b *bbPool.ByteBuffer
}

// udpConn 用于打包 UDP 连接。
type udpConn struct {
	c *conn
}

// openConn 表示待打开连接及其回调。
type openConn struct {
	c  *conn
	cb func()
}

// conn 表示一个连接对象。
type conn struct {
	pc            net.PacketConn       // UDP 使用
	ctx           any                  // 用户定义的上下文
	loop          *eventloop           // 所属事件循环
	buffer        *bbPool.ByteBuffer   // 入站数据的临时缓冲，复用内存
	cache         []byte               // 入站数据的临时缓存
	rawConn       net.Conn             // 原始连接
	localAddr     net.Addr             // 本地地址
	remoteAddr    net.Addr             // 远端地址
	inboundBuffer xelastic.XRingBuffer // 入站环形缓冲
}

// packTCPConn 打包 TCP 读到的数据。
func packTCPConn(c *conn, buf []byte) *tcpConn {
	b := bbPool.Get()
	_, _ = b.Write(buf)
	return &tcpConn{c: c, b: b}
}

// unpackTCPConn 解包 TCP 数据并追加到连接缓冲中。
func unpackTCPConn(tc *tcpConn) *conn {
	if tc.c.buffer == nil { // 连接已关闭
		return nil
	}
	_, _ = tc.c.buffer.Write(tc.b.B)
	bbPool.Put(tc.b)
	tc.b = nil
	return tc.c
}

// packUDPConn 打包 UDP 数据。
func packUDPConn(c *conn, buf []byte) *udpConn {
	_, _ = c.buffer.Write(buf)
	return &udpConn{c: c}
}

// newStreamConn 创建 TCP/Unix 风格连接。
func newStreamConn(el *eventloop, nc net.Conn, ctx any) (c *conn) {
	return &conn{
		ctx:        ctx,
		loop:       el,
		buffer:     bbPool.Get(),
		rawConn:    nc,
		localAddr:  nc.LocalAddr(),
		remoteAddr: nc.RemoteAddr(),
	}
}

// release 释放连接占用资源。
func (c *conn) release() {
	c.ctx = nil
	c.localAddr = nil

	if c.rawConn != nil {
		c.rawConn = nil
		c.remoteAddr = nil
	}

	c.inboundBuffer.Done()

	if c.buffer != nil {
		bbPool.Put(c.buffer)
		c.buffer = nil
	}
}

// newUDPConn 创建 UDP 连接。
func newUDPConn(el *eventloop, pc net.PacketConn, rc net.Conn, localAddr, remoteAddr net.Addr, ctx any) *conn {
	return &conn{
		ctx:        ctx,
		pc:         pc,
		rawConn:    rc,
		loop:       el,
		buffer:     bbPool.Get(),
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

// resetBuffer 重置入站缓冲区。
func (c *conn) resetBuffer() {
	c.buffer.Reset()
	c.inboundBuffer.Reset()
	c.inboundBuffer.Done()
}

// Read 从连接缓冲中读取数据。
func (c *conn) Read(p []byte) (n int, err error) {
	if c.inboundBuffer.IsEmpty() {
		n = copy(p, c.buffer.B)
		c.buffer.B = c.buffer.B[n:]
		if n == 0 && len(p) > 0 {
			err = io.ErrShortBuffer
		}
		return
	}

	n, _ = c.inboundBuffer.Read(p)
	if n == len(p) {
		return
	}

	m := copy(p[n:], c.buffer.B)
	n += m
	c.buffer.B = c.buffer.B[m:]
	return
}

// Next 读取接下来 n 字节数据，n<=0 表示全部读取。
func (c *conn) Next(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + c.buffer.Len(); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}

	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer.B[:n]
		c.buffer.B = c.buffer.B[n:]
		return
	}

	buf = bsPool.Get(n)
	_, err = c.Read(buf)
	return
}

// Peek 查看接下来 n 字节数据但不消费，n<=0 表示查看全部。
func (c *conn) Peek(n int) (buf []byte, err error) {
	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + c.buffer.Len(); n > totalLen {
		return nil, io.ErrShortBuffer
	} else if n <= 0 {
		n = totalLen
	}

	if c.inboundBuffer.IsEmpty() {
		return c.buffer.B[:n], err
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
	buf = append(buf, c.buffer.B[:remaining]...)
	c.cache = buf
	return
}

// Discard 丢弃接下来 n 字节数据，n<=0 表示全部丢弃。
func (c *conn) Discard(n int) (int, error) {
	if len(c.cache) > 0 {
		bsPool.Put(c.cache)
		c.cache = nil
	}

	inBufferLen := c.inboundBuffer.Buffered()
	if totalLen := inBufferLen + c.buffer.Len(); n >= totalLen || n <= 0 {
		c.resetBuffer()
		return totalLen, nil
	}

	if c.inboundBuffer.IsEmpty() {
		c.buffer.B = c.buffer.B[n:]
		return n, nil
	}

	discarded, _ := c.inboundBuffer.Discard(n)
	if discarded < inBufferLen {
		return discarded, nil
	}

	remaining := n - inBufferLen
	c.buffer.B = c.buffer.B[remaining:]
	return n, nil
}

// Write 写数据到连接。
func (c *conn) Write(p []byte) (int, error) {
	if c.rawConn == nil && c.pc == nil {
		return 0, net.ErrClosed
	}

	if c.rawConn != nil {
		return c.rawConn.Write(p)
	}

	return c.pc.WriteTo(p, c.remoteAddr)
}

// SendTo 向指定地址发送数据，仅 UDP 支持。
func (c *conn) SendTo(p []byte, addr net.Addr) (int, error) {
	if c.pc == nil {
		return 0, xerrors.ErrUnsupportedOp
	}
	if addr == nil {
		return 0, xerrors.ErrInvalidNetworkAddress
	}
	return c.pc.WriteTo(p, addr)
}

// Writev 批量写数据。
// Windows 下底层不支持原生 writev，这里会先合并再写。
func (c *conn) Writev(bs [][]byte) (int, error) {
	if c.pc != nil { // UDP 不支持
		return 0, xerrors.ErrUnsupportedOp
	}

	if c.rawConn != nil {
		bb := bbPool.Get()
		defer bbPool.Put(bb)

		for i := range bs {
			_, _ = bb.Write(bs[i])
		}
		return c.rawConn.Write(bb.Bytes())
	}

	return 0, net.ErrClosed
}

// ReadFrom 从 Reader 读取并写入连接。
func (c *conn) ReadFrom(r io.Reader) (int64, error) {
	if c.rawConn != nil {
		return io.Copy(c.rawConn, r)
	}
	return 0, net.ErrClosed
}

// WriteTo 将缓冲中的数据写到 Writer。
func (c *conn) WriteTo(w io.Writer) (n int64, err error) {
	if !c.inboundBuffer.IsEmpty() {
		if n, err = c.inboundBuffer.WriteTo(w); err != nil {
			return
		}
	}

	if c.buffer == nil {
		return 0, nil
	}

	defer c.buffer.Reset()
	return c.buffer.WriteTo(w)
}

// Flush 在 Windows 版本中无需额外刷新。
func (c *conn) Flush() error {
	return nil
}

// InboundBuffered 返回入站缓冲字节数。
func (c *conn) InboundBuffered() int {
	if c.buffer == nil {
		return 0
	}
	return c.inboundBuffer.Buffered() + c.buffer.Len()
}

// OutboundBuffered Windows 版本未维护独立出站缓冲。
func (c *conn) OutboundBuffered() int {
	return 0
}

func (c *conn) Context() any         { return c.ctx }
func (c *conn) SetContext(ctx any)   { c.ctx = ctx }
func (c *conn) LocalAddr() net.Addr  { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr { return c.remoteAddr }

// Fd 获取底层 fd/handle。
func (c *conn) Fd() (fd int) {
	if c.rawConn == nil {
		return -1
	}

	rc, err := c.rawConn.(syscall.Conn).SyscallConn()
	if err != nil {
		return -1
	}

	if err := rc.Control(func(i uintptr) {
		fd = int(i)
	}); err != nil {
		return -1
	}
	return
}

// Dup 复制底层 handle。
func (c *conn) Dup() (fd int, err error) {
	if c.rawConn == nil && c.pc == nil {
		return -1, net.ErrClosed
	}

	var (
		sc syscall.Conn
		ok bool
	)

	if c.rawConn != nil {
		sc, ok = c.rawConn.(syscall.Conn)
	} else {
		sc, ok = c.pc.(syscall.Conn)
	}

	if !ok {
		return -1, xerrors.New("failed to convert net.Conn to syscall.Conn")
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return -1, xerrors.New("failed to get syscall.RawConn from net.Conn")
	}

	var dupHandle windows.Handle
	e := rc.Control(func(fd uintptr) {
		process := windows.CurrentProcess()
		err = windows.DuplicateHandle(
			process,
			windows.Handle(fd),
			process,
			&dupHandle,
			0,
			true,
			windows.DUPLICATE_SAME_ACCESS,
		)
	})
	if err != nil {
		return -1, err
	}
	if e != nil {
		return -1, e
	}

	return int(dupHandle), nil
}

// SetReadBuffer 设置读缓冲区大小。
func (c *conn) SetReadBuffer(bytes int) error {
	if c.rawConn == nil && c.pc == nil {
		return net.ErrClosed
	}

	if c.rawConn != nil {
		return c.rawConn.(interface{ SetReadBuffer(int) error }).SetReadBuffer(bytes)
	}
	return c.pc.(interface{ SetReadBuffer(int) error }).SetReadBuffer(bytes)
}

// SetWriteBuffer 设置写缓冲区大小。
func (c *conn) SetWriteBuffer(bytes int) error {
	if c.rawConn == nil && c.pc == nil {
		return net.ErrClosed
	}

	if c.rawConn != nil {
		return c.rawConn.(interface{ SetWriteBuffer(int) error }).SetWriteBuffer(bytes)
	}
	return c.pc.(interface{ SetWriteBuffer(int) error }).SetWriteBuffer(bytes)
}

// SetLinger 设置 linger，仅 TCP 支持。
func (c *conn) SetLinger(sec int) error {
	if c.rawConn == nil {
		return net.ErrClosed
	}

	tc, ok := c.rawConn.(*net.TCPConn)
	if !ok {
		return xerrors.ErrUnsupportedOp
	}
	return tc.SetLinger(sec)
}

// SetNoDelay 设置 TCP_NODELAY，仅 TCP 支持。
func (c *conn) SetNoDelay(noDelay bool) error {
	if c.rawConn == nil {
		return net.ErrClosed
	}

	tc, ok := c.rawConn.(*net.TCPConn)
	if !ok {
		return xerrors.ErrUnsupportedOp
	}
	return tc.SetNoDelay(noDelay)
}

// SetKeepAlivePeriod 设置 keepalive 周期。
func (c *conn) SetKeepAlivePeriod(d time.Duration) error {
	return c.SetKeepAlive(d > 0, d, d/5, 5)
}

// SetKeepAlive 设置 TCP keepalive，仅 TCP 支持。
func (c *conn) SetKeepAlive(enabled bool, idle, intvl time.Duration, cnt int) error {
	if c.rawConn == nil && c.pc == nil {
		return net.ErrClosed
	}
	if c.pc != nil {
		return xerrors.ErrUnsupportedOp
	}

	tc, ok := c.rawConn.(*net.TCPConn)
	if !ok {
		return xerrors.ErrUnsupportedOp
	}

	if enabled && (idle <= 0 || intvl <= 0 || cnt <= 0) {
		return xerrors.New("invalid time duration")
	}

	if err := tc.SetKeepAlive(enabled); err != nil {
		return err
	}
	if !enabled {
		return nil
	}

	if err := tc.SetKeepAlivePeriod(idle); err != nil {
		return err
	}

	if err := windows.SetsockoptInt(
		windows.Handle(c.Fd()),
		windows.IPPROTO_TCP,
		windows.TCP_KEEPINTVL,
		int(intvl.Seconds()),
	); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}

	if err := windows.SetsockoptInt(
		windows.Handle(c.Fd()),
		windows.IPPROTO_TCP,
		windows.TCP_KEEPCNT,
		cnt,
	); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}

	return nil
}

// Gfd 返回一个未初始化的 GFD，仅为兼容保留，不应在 Windows 下使用。
// func (c *conn) Gfd() gfd.GFD { return gfd.GFD{} }

// AsyncWrite 异步写数据。
func (c *conn) AsyncWrite(buf []byte, cb AsyncCallback) error {
	fn := func() error {
		_, err := c.Write(buf)
		if cb != nil {
			_ = cb(c, err)
		}
		return err
	}

	var err error
	select {
	case c.loop.ch <- fn:
	default:
		// 当 event-loop 通道已满时，异步转交，避免阻塞 event-loop。
		err = xgoroutine.DefaultWorkerPool.Submit(func() {
			c.loop.ch <- fn
		})
	}

	return err
}

// AsyncWritev 异步批量写数据。
func (c *conn) AsyncWritev(bs [][]byte, cb AsyncCallback) error {
	if c.pc != nil {
		return xerrors.ErrUnsupportedOp
	}

	buf := bbPool.Get()
	for _, b := range bs {
		_, _ = buf.Write(b)
	}

	return c.AsyncWrite(buf.Bytes(), func(c Conn, err error) error {
		defer bbPool.Put(buf)
		if cb == nil {
			return err
		}
		return cb(c, err)
	})
}

// Wake 异步唤醒连接。
func (c *conn) Wake(cb AsyncCallback) (err error) {
	wakeFn := func() (err error) {
		err = c.loop.wake(c)
		if cb != nil {
			_ = cb(c, err)
		}
		return
	}

	select {
	case c.loop.ch <- wakeFn:
	default:
		// 当 event-loop 通道已满时，异步转交，避免阻塞 event-loop。
		err = xgoroutine.DefaultWorkerPool.Submit(func() {
			c.loop.ch <- wakeFn
		})
	}

	return
}

// Close 异步关闭连接。
func (c *conn) Close() (err error) {
	closeFn := func() error {
		return c.loop.close(c, nil)
	}

	select {
	case c.loop.ch <- closeFn:
	default:
		// 当 event-loop 通道已满时，异步转交，避免阻塞 event-loop。
		err = xgoroutine.DefaultWorkerPool.Submit(func() {
			c.loop.ch <- closeFn
		})
	}

	return
}

// CloseWithCallback 异步关闭连接并执行回调。
func (c *conn) CloseWithCallback(cb AsyncCallback) (err error) {
	closeFn := func() (err error) {
		err = c.loop.close(c, nil)
		if cb != nil {
			_ = cb(c, err)
		}
		return
	}

	select {
	case c.loop.ch <- closeFn:
	default:
		// 当 event-loop 通道已满时，异步转交，避免阻塞 event-loop。
		err = xgoroutine.DefaultWorkerPool.Submit(func() {
			c.loop.ch <- closeFn
		})
	}

	return
}

// EventLoop 返回所属事件循环。
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
