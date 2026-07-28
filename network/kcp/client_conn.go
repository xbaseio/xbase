package kcp

import (
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"github.com/xtaci/kcp-go/v5"
	"go.uber.org/zap"
)

type clientConn struct {
	id     int64        // 连接ID
	uid    atomic.Int64 // 用户ID
	attr   *attr        // 连接属性
	conn   atomic.Pointer[kcp.UDPSession]
	state  atomic.Int32 // 连接状态
	client *client      // 客户端

	chWrite chan chWrite  // 写入队列
	done    chan struct{} // 写入完成信号，使用 close 通知
	close   chan struct{} // 关闭信号

	doneOnce  sync.Once
	closeOnce sync.Once
}

var _ network.Conn = &clientConn{}

func newClientConn(client *client, id int64, conn *kcp.UDPSession) network.Conn {
	c := &clientConn{
		id:      id,
		attr:    &attr{},
		client:  client,
		chWrite: make(chan chWrite, 4096),
		done:    make(chan struct{}),
		close:   make(chan struct{}),
	}

	c.conn.Store(conn)
	c.state.Store(int32(network.ConnOpened))
	c.applyKCPOptions(conn)

	xcall.Go(c.read)
	xcall.Go(c.write)

	if c.client.connectHandler != nil {
		c.client.connectHandler(c)
	}

	return c
}

func (c *clientConn) applyKCPOptions(conn *kcp.UDPSession) {
	if conn == nil {
		return
	}

	if c.client.opts.mtu > 0 {
		conn.SetMtu(c.client.opts.mtu)
	}

	if len(c.client.opts.noDelay) == 4 {
		conn.SetNoDelay(
			c.client.opts.noDelay[0],
			c.client.opts.noDelay[1],
			c.client.opts.noDelay[2],
			c.client.opts.noDelay[3],
		)
	}

	if c.client.opts.ackNoDelay {
		conn.SetACKNoDelay(c.client.opts.ackNoDelay)
	}

	if c.client.opts.writeDelay {
		conn.SetWriteDelay(c.client.opts.writeDelay)
	}

	if len(c.client.opts.windowSize) == 2 {
		conn.SetWindowSize(
			c.client.opts.windowSize[0],
			c.client.opts.windowSize[1],
		)
	}

	if c.client.opts.readBuffer > 0 {
		conn.SetReadBuffer(c.client.opts.readBuffer)
	}

	if c.client.opts.writeBuffer > 0 {
		conn.SetWriteBuffer(c.client.opts.writeBuffer)
	}
}

// ID 获取连接ID
func (c *clientConn) ID() int64 {
	return c.id
}

// UID 获取用户ID
func (c *clientConn) UID() int64 {
	return c.uid.Load()
}

// Attr 获取属性接口
func (c *clientConn) Attr() network.Attr {
	return c.attr
}

// Bind 绑定用户ID
func (c *clientConn) Bind(uid int64) {
	c.uid.Store(uid)
}

// Unbind 解绑用户ID
func (c *clientConn) Unbind() {
	c.uid.Store(0)
}

// Send 发送消息。
// 无锁版里 Send 不直接写 conn，而是进入写队列。
// 真正写 KCP 的地方只有 write() 一个 goroutine。
func (c *clientConn) Send(msg []byte) error {
	return c.enqueueWrite(msg)
}

// Push 发送消息（异步）
func (c *clientConn) Push(msg []byte) error {
	return c.enqueueWrite(msg)
}

func (c *clientConn) enqueueWrite(msg []byte) error {
	if len(msg) == 0 {
		return nil
	}

	if c.State() != network.ConnOpened {
		return c.checkState()
	}

	if c.getConn() == nil {
		return xerrors.ErrConnectionClosed
	}

	select {
	case c.chWrite <- chWrite{typ: dataPacket, msg: msg}:
		// 无锁版本不保证关闭瞬间的强一致。
		// 入队后如果连接状态已经变化，返回关闭错误。
		if c.State() != network.ConnOpened {
			return xerrors.ErrConnectionClosed
		}
		return nil

	case <-c.close:
		return xerrors.ErrConnectionClosed
	}
}

// State 获取连接状态
func (c *clientConn) State() network.ConnState {
	return network.ConnState(c.state.Load())
}

// Close 关闭连接
func (c *clientConn) Close(force ...bool) error {
	if len(force) > 0 && force[0] {
		return c.forceClose()
	}

	return c.graceClose()
}

// LocalIP 获取本地IP
func (c *clientConn) LocalIP() (string, error) {
	addr, err := c.LocalAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

// LocalAddr 获取本地地址
func (c *clientConn) LocalAddr() (net.Addr, error) {
	if err := c.checkState(); err != nil {
		return nil, err
	}

	conn := c.getConn()
	if conn == nil {
		return nil, xerrors.ErrConnectionClosed
	}

	return conn.LocalAddr(), nil
}

// RemoteIP 获取远端IP
func (c *clientConn) RemoteIP() (string, error) {
	addr, err := c.RemoteAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

// RemoteAddr 获取远端地址
func (c *clientConn) RemoteAddr() (net.Addr, error) {
	if err := c.checkState(); err != nil {
		return nil, err
	}

	conn := c.getConn()
	if conn == nil {
		return nil, xerrors.ErrConnectionClosed
	}

	return conn.RemoteAddr(), nil
}

// 检测连接状态
func (c *clientConn) checkState() error {
	switch c.State() {
	case network.ConnHanged:
		return xerrors.ErrConnectionHanged
	case network.ConnClosed:
		return xerrors.ErrConnectionClosed
	default:
		return nil
	}
}

// 优雅关闭
func (c *clientConn) graceClose() error {
	if !c.state.CompareAndSwap(int32(network.ConnOpened), int32(network.ConnHanged)) {
		switch c.State() {
		case network.ConnHanged:
			<-c.done
			return c.finishGraceClose()

		case network.ConnClosed:
			return nil

		default:
			return xerrors.ErrConnectionNotOpened
		}
	}

	if c.getConn() == nil {
		return c.finishGraceClose()
	}

	select {
	case c.chWrite <- chWrite{typ: closeSig}:

	case <-c.close:
		return nil
	}

	<-c.done
	return c.finishGraceClose()
}

func (c *clientConn) finishGraceClose() error {
	if c.state.CompareAndSwap(int32(network.ConnHanged), int32(network.ConnClosed)) {
		return c.doClose()
	}

	if c.State() == network.ConnClosed {
		return nil
	}

	return xerrors.ErrConnectionNotHanged
}

// 强制关闭
func (c *clientConn) forceClose() error {
	for {
		state := c.State()

		switch state {
		case network.ConnClosed:
			return xerrors.ErrConnectionClosed

		case network.ConnOpened, network.ConnHanged:
			if c.state.CompareAndSwap(int32(state), int32(network.ConnClosed)) {
				return c.doClose()
			}

		default:
			return xerrors.ErrConnectionClosed
		}
	}
}

// 执行关闭操作
func (c *clientConn) doClose() error {
	var closeErr error

	c.closeOnce.Do(func() {
		// 先关闭 close，让 Push/graceClose/read/write 里的 select 尽快退出。
		close(c.close)

		// done 用 close 通知，避免无缓冲 channel 发送阻塞。
		c.notifyDone()

		conn := c.conn.Swap(nil)
		if conn != nil {
			closeErr = conn.Close()
		} else {
			closeErr = xerrors.ErrConnectionClosed
		}

		if c.client.disconnectHandler != nil {
			c.client.disconnectHandler(c)
		}
	})

	return closeErr
}

// 读取消息
func (c *clientConn) read() {
	conn := c.getConn()
	closeCh := c.close

	if conn == nil {
		_ = c.forceClose()
		return
	}

	for {
		select {
		case <-closeCh:
			return
		default:
		}

		data, err := packet.ReadMessage(conn)
		if err != nil {
			_ = c.forceClose()
			return
		}

		switch c.State() {
		case network.ConnHanged:
			continue

		case network.ConnClosed:
			return

		default:
			// ignore
		}

		// ignore empty packet
		if len(data) == 0 {
			continue
		}

		if c.client.receiveHandler != nil {
			c.client.receiveHandler(c, data)
		}
	}
}

// 写入消息
func (c *clientConn) write() {
	closeCh := c.close
	chWrite := c.chWrite

	for {
		select {
		case <-closeCh:
			return

		case r, ok := <-chWrite:
			if !ok {
				c.notifyDone()
				return
			}

			if r.typ == closeSig {
				c.notifyDone()
				return
			}

			if r.typ != dataPacket {
				continue
			}

			if len(r.msg) == 0 {
				continue
			}

			if c.isClosed() {
				return
			}

			conn := c.getConn()
			if conn == nil {
				_ = c.forceClose()
				return
			}

			if err := c.writeFull(conn, r.msg); err != nil {
				xlog.Logger().Error("write data message error", zap.Error(err))
				_ = c.forceClose()
				return
			}
		}
	}
}

// 是否已关闭
func (c *clientConn) isClosed() bool {
	return c.State() == network.ConnClosed
}

func (c *clientConn) getConn() *kcp.UDPSession {
	return c.conn.Load()
}

func (c *clientConn) notifyDone() {
	c.doneOnce.Do(func() {
		close(c.done)
	})
}

func (c *clientConn) writeFull(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}

		if n <= 0 {
			return io.ErrShortWrite
		}

		data = data[n:]
	}

	return nil
}
