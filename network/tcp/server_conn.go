package tcp

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/utils/xtime"
	"github.com/xbaseio/xbase/xerrors"
)

type serverConnBox struct {
	conn net.Conn
}

type serverConn struct {
	id      int64          // 连接ID
	uid     atomic.Int64   // 用户ID
	attr    *attr          // 连接属性
	state   atomic.Int32   // 连接状态
	connMgr *serverConnMgr // 连接管理

	conn atomic.Pointer[serverConnBox] // TCP源连接

	chWrite chan chWrite  // 写入队列
	done    chan struct{} // 写入完成信号，使用 close 通知
	close   chan struct{} // 关闭信号

	doneOnce  sync.Once
	closeOnce sync.Once

	// generation 用来防止 serverConn 复用后，旧 read/write/timer 误关新连接
	generation atomic.Int64

	lastHeartbeatTime atomic.Int64 // 上次心跳时间
	authorizeTimer    atomic.Value // 授权定时器
}

var _ network.Conn = &serverConn{}

// ID 获取连接ID
func (c *serverConn) ID() int64 {
	return c.id
}

// UID 获取用户ID
func (c *serverConn) UID() int64 {
	return c.uid.Load()
}

// Attr 获取属性接口
func (c *serverConn) Attr() network.Attr {
	return c.attr
}

// Bind 绑定用户ID
func (c *serverConn) Bind(uid int64) {
	c.uid.Store(uid)
	c.uncheckAuthorize()
}

// Unbind 解绑用户ID
func (c *serverConn) Unbind() {
	c.uid.Store(0)
	c.checkAuthorize()
}

// Send 发送消息。
// 去锁版里 Send 不直接写 conn，而是进入写队列。
// 真正写 TCP 的地方只有 write() 一个 goroutine。
func (c *serverConn) Send(msg []byte) error {
	return c.enqueueWrite(msg)
}

// Push 发送消息（异步）
func (c *serverConn) Push(msg []byte) error {
	return c.enqueueWrite(msg)
}

func (c *serverConn) enqueueWrite(msg []byte) error {
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
		// 去锁版本不保证关闭瞬间的强一致。
		// 入队后如果连接状态已经变化，返回对应错误。
		if c.State() != network.ConnOpened {
			return c.checkState()
		}
		return nil

	case <-c.close:
		return xerrors.ErrConnectionClosed
	}
}

// State 获取连接状态
func (c *serverConn) State() network.ConnState {
	return network.ConnState(c.state.Load())
}

// Close 关闭连接
func (c *serverConn) Close(force ...bool) error {
	if len(force) > 0 && force[0] {
		return c.forceClose(true)
	}

	return c.graceClose(true)
}

// LocalIP 获取本地IP
func (c *serverConn) LocalIP() (string, error) {
	addr, err := c.LocalAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

// LocalAddr 获取本地地址
func (c *serverConn) LocalAddr() (net.Addr, error) {
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
func (c *serverConn) RemoteIP() (string, error) {
	addr, err := c.RemoteAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

// RemoteAddr 获取远端地址
func (c *serverConn) RemoteAddr() (net.Addr, error) {
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
func (c *serverConn) checkState() error {
	switch c.State() {
	case network.ConnHanged:
		return xerrors.ErrConnectionHanged
	case network.ConnClosed:
		return xerrors.ErrConnectionClosed
	default:
		return nil
	}
}

// 授权检查
func (c *serverConn) checkAuthorize() {
	if c.connMgr.server.opts.authorizeTimeout <= 0 {
		return
	}

	gen := c.generation.Load()

	timer := c.authorizeTimer.Swap(time.AfterFunc(c.connMgr.server.opts.authorizeTimeout, func() {
		// serverConn 可能复用，旧 timer 不能误关新连接
		if c.generation.Load() != gen {
			return
		}

		if c.UID() != 0 {
			return
		}

		_ = c.forceCloseIfCurrent(gen, true)
	}))

	if t, ok := timer.(*time.Timer); ok && t != nil {
		t.Stop()
	}
}

// 取消授权检查
func (c *serverConn) uncheckAuthorize() {
	if c.connMgr.server.opts.authorizeTimeout <= 0 {
		return
	}

	timer := c.authorizeTimer.Swap((*time.Timer)(nil))
	if t, ok := timer.(*time.Timer); ok && t != nil {
		t.Stop()
	}
}

// 初始化连接
func (c *serverConn) init(cm *serverConnMgr, id int64, conn net.Conn) {
	// 如果 serverConn 会复用，Once/channel 必须重新初始化
	c.doneOnce = sync.Once{}
	c.closeOnce = sync.Once{}

	c.generation.Add(1)

	c.id = id
	c.uid.Store(0)
	c.attr = &attr{}
	c.state.Store(int32(network.ConnOpened))
	c.conn.Store(&serverConnBox{conn: conn})
	c.connMgr = cm
	c.chWrite = make(chan chWrite, 4096)
	c.done = make(chan struct{})
	c.close = make(chan struct{})
	c.lastHeartbeatTime.Store(xtime.Now().UnixNano())
	c.authorizeTimer.Store((*time.Timer)(nil))

	xcall.Go(c.read)
	xcall.Go(c.write)

	c.checkAuthorize()

	if c.connMgr.server.connectHandler != nil {
		c.connMgr.server.connectHandler(c)
	}
}

// 重置连接
func (c *serverConn) reset() {
	c.attr = nil
}

// 优雅关闭
func (c *serverConn) graceClose(isNeedRecycle bool) error {
	if !c.state.CompareAndSwap(int32(network.ConnOpened), int32(network.ConnHanged)) {
		switch c.State() {
		case network.ConnHanged:
			<-c.done
			return c.finishGraceClose(isNeedRecycle)

		case network.ConnClosed:
			return nil

		default:
			return xerrors.ErrConnectionNotOpened
		}
	}

	c.uncheckAuthorize()

	if c.getConn() == nil {
		return c.finishGraceClose(isNeedRecycle)
	}

	select {
	case c.chWrite <- chWrite{typ: closeSig}:

	case <-c.close:
		return nil
	}

	<-c.done
	return c.finishGraceClose(isNeedRecycle)
}

func (c *serverConn) finishGraceClose(isNeedRecycle bool) error {
	if c.state.CompareAndSwap(int32(network.ConnHanged), int32(network.ConnClosed)) {
		return c.doClose(isNeedRecycle)
	}

	if c.State() == network.ConnClosed {
		return nil
	}

	return xerrors.ErrConnectionNotHanged
}

// 强制关闭
func (c *serverConn) forceClose(isNeedRecycle bool) error {
	for {
		state := c.State()

		switch state {
		case network.ConnClosed:
			return xerrors.ErrConnectionClosed

		case network.ConnOpened, network.ConnHanged:
			if c.state.CompareAndSwap(int32(state), int32(network.ConnClosed)) {
				c.uncheckAuthorize()
				return c.doClose(isNeedRecycle)
			}

		default:
			return xerrors.ErrConnectionClosed
		}
	}
}

func (c *serverConn) forceCloseIfCurrent(gen int64, isNeedRecycle bool) error {
	for {
		if c.generation.Load() != gen {
			return nil
		}

		state := c.State()

		switch state {
		case network.ConnClosed:
			return xerrors.ErrConnectionClosed

		case network.ConnOpened, network.ConnHanged:
			if c.generation.Load() != gen {
				return nil
			}

			if c.state.CompareAndSwap(int32(state), int32(network.ConnClosed)) {
				c.uncheckAuthorize()
				return c.doCloseIfCurrent(gen, isNeedRecycle)
			}

		default:
			return xerrors.ErrConnectionClosed
		}
	}
}

func (c *serverConn) doCloseIfCurrent(gen int64, isNeedRecycle bool) error {
	if c.generation.Load() != gen {
		return nil
	}

	return c.doClose(isNeedRecycle)
}

// 执行关闭操作
func (c *serverConn) doClose(isNeedRecycle bool) error {
	var closeErr error

	c.closeOnce.Do(func() {
		c.uncheckAuthorize()

		// 先关闭 close，让 Push/graceClose/read/write 里的 select 尽快退出。
		close(c.close)

		// done 用 close 通知，避免无缓冲 channel 发送阻塞。
		c.notifyDone()

		box := c.conn.Swap(nil)
		if box != nil && box.conn != nil {
			closeErr = box.conn.Close()
		} else {
			closeErr = xerrors.ErrConnectionClosed
		}

		if c.connMgr.server.disconnectHandler != nil {
			c.connMgr.server.disconnectHandler(c)
		}

		if isNeedRecycle && box != nil && box.conn != nil {
			c.connMgr.recycle(box.conn)
		}
	})

	return closeErr
}

// 读取消息
func (c *serverConn) read() {
	gen := c.generation.Load()
	conn := c.getConn()
	closeCh := c.close

	if conn == nil {
		_ = c.forceCloseIfCurrent(gen, true)
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
			_ = c.forceCloseIfCurrent(gen, true)
			return
		}

		// 当前 goroutine 属于旧连接，连接对象已经被复用，直接退出
		if c.generation.Load() != gen {
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

		if c.connMgr.server.receiveHandler != nil {
			c.connMgr.server.receiveHandler(c, data)
		}
	}
}

// 写入消息
func (c *serverConn) write() {
	gen := c.generation.Load()
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

			if c.generation.Load() != gen {
				return
			}

			if c.isClosed() {
				return
			}

			conn := c.getConn()
			if conn == nil {
				_ = c.forceCloseIfCurrent(gen, true)
				return
			}

			if err := c.writeFull(conn, r.msg); err != nil {
				log.Errorf("write data message error: %v", err)
				_ = c.forceCloseIfCurrent(gen, true)
				return
			}
		}
	}
}

// 是否已关闭
func (c *serverConn) isClosed() bool {
	return c.State() == network.ConnClosed
}

func (c *serverConn) getConn() net.Conn {
	box := c.conn.Load()
	if box == nil {
		return nil
	}

	return box.conn
}

func (c *serverConn) notifyDone() {
	c.doneOnce.Do(func() {
		close(c.done)
	})
}

func (c *serverConn) writeFull(conn net.Conn, data []byte) error {
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
