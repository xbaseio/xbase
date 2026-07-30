package ws

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type serverConn struct {
	id      int64          // 连接ID
	uid     atomic.Int64   // 用户ID
	attr    *attr          // 连接属性
	state   atomic.Int32   // 连接状态
	connMgr *serverConnMgr // 连接管理

	rw     sync.RWMutex // conn 保护锁
	sendMu sync.Mutex   // 发送锁，保护 Send/Push/closeSig 和 channel close 的并发安全

	conn        *websocket.Conn // WS源连接
	recvQ       chan []byte     // 收包队列
	chLowWrite  chan chWrite    // 低优先级队列
	chHighWrite chan chWrite    // 高优先级队列
	done        chan struct{}   // 写入完成信号，使用 close 通知
	close       chan struct{}   // 关闭信号

	doneOnce  sync.Once
	closeOnce sync.Once

	authorizeTimer atomic.Value // 授权定时器
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
// 注意：gorilla/websocket 不允许并发写，这里仍然走高优先级写队列。
func (c *serverConn) Send(msg []byte) error {
	return c.enqueueWrite(c.chHighWrite, msg)
}

// Push 发送消息（异步）
func (c *serverConn) Push(msg []byte) error {
	return c.enqueueWrite(c.chLowWrite, msg)
}

func (c *serverConn) enqueueWrite(ch chan chWrite, msg []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	if err := c.checkState(); err != nil {
		return err
	}

	if len(msg) == 0 {
		return nil
	}

	if c.getConn() == nil {
		return xerrors.ErrConnectionClosed
	}

	select {
	case <-c.close:
		return xerrors.ErrConnectionClosed

	default:
	}

	select {
	case ch <- chWrite{typ: dataPacket, msg: msg}:
		if c.isClosed() {
			return xerrors.ErrConnectionClosed
		}
		return nil

	case <-c.close:
		return xerrors.ErrConnectionClosed

	default:
	}

	timer := time.NewTimer(network.DefaultWriteEnqueueTimeout)
	connID := c.ID()
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	select {
	case ch <- chWrite{typ: dataPacket, msg: msg}:
		if c.isClosed() {
			return xerrors.ErrConnectionClosed
		}
		return nil

	case <-c.close:
		return xerrors.ErrConnectionClosed

	case <-timer.C:
		go func() {
			if c.ID() == connID {
				_ = c.forceClose(true)
			}
		}()
		return xerrors.ErrWriteQueueTimeout
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

// 初始化连接
func (c *serverConn) init(cm *serverConnMgr, id int64, conn *websocket.Conn) {
	// 如果 serverConn 会被复用，锁、Once、channel 必须重新初始化。
	c.rw = sync.RWMutex{}
	c.sendMu = sync.Mutex{}
	c.doneOnce = sync.Once{}
	c.closeOnce = sync.Once{}

	c.id = id
	c.uid.Store(0)
	c.attr = &attr{}
	c.state.Store(int32(network.ConnOpened))
	c.conn = conn
	c.connMgr = cm
	c.recvQ = make(chan []byte, network.DefaultRecvQueueSize)
	c.chLowWrite = make(chan chWrite, network.DefaultWriteQueueSize)
	c.chHighWrite = make(chan chWrite, network.DefaultWriteQueueSize/4)
	c.done = make(chan struct{})
	c.close = make(chan struct{})
	c.authorizeTimer.Store((*time.Timer)(nil))

	xcall.Go(func() { c.receiveLoop(c.ID(), c.recvQ) })
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

	// 连接对象可能被复用，timer 回调必须绑定当前连接ID，
	// 防止旧连接的 timer 误关新连接。
	connID := c.ID()

	timer := c.authorizeTimer.Swap(time.AfterFunc(c.connMgr.server.opts.authorizeTimeout, func() {
		if c.ID() != connID {
			return
		}

		if c.UID() != 0 {
			return
		}

		xlog.Logger().Warn("ws authorize timeout, close conn",
			zap.Int64("cid", connID),
			zap.Duration("timeout", c.connMgr.server.opts.authorizeTimeout),
		)
		_ = c.forceCloseIfCurrent(connID, true)
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

	// 等待已经进入 Send/Push 的请求完成。
	// ConnHanged 后新的 Send/Push 会被拒绝。
	c.sendMu.Lock()

	if c.State() == network.ConnClosed {
		c.sendMu.Unlock()
		return nil
	}

	if c.getConn() == nil {
		c.sendMu.Unlock()
		return c.finishGraceClose(isNeedRecycle)
	}

	select {
	case c.chLowWrite <- chWrite{typ: closeSig}:
		c.sendMu.Unlock()

	case <-c.close:
		c.sendMu.Unlock()
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

func (c *serverConn) forceCloseIfCurrent(connID int64, isNeedRecycle bool) error {
	if c.ID() != connID {
		return nil
	}

	return c.forceClose(isNeedRecycle)
}

// 执行关闭操作
func (c *serverConn) doClose(isNeedRecycle bool) error {
	var closeErr error

	c.closeOnce.Do(func() {
		c.uncheckAuthorize()

		// 先关闭 close，让 Send/Push/graceClose/read/write 中的 select 退出。
		close(c.close)

		// done 使用 close 通知，避免无缓冲 channel 发送卡死。
		c.notifyDone()

		if c.recvQ != nil {
			close(c.recvQ)
		}

		// 等待正在 Send/Push 的 goroutine 退出，然后再关闭写队列。
		// 这样可以避免 close(channel) 和 channel <- value 并发导致 panic。
		c.sendMu.Lock()
		close(c.chHighWrite)
		close(c.chLowWrite)
		c.sendMu.Unlock()

		c.rw.Lock()
		conn := c.conn
		c.conn = nil
		c.rw.Unlock()

		if conn != nil {
			closeErr = conn.Close()
		} else {
			closeErr = xerrors.ErrConnectionClosed
		}

		if c.connMgr.server.disconnectHandler != nil {
			c.connMgr.server.disconnectHandler(c)
		}

		if isNeedRecycle && conn != nil {
			c.connMgr.recycle(conn)
		}
	})

	return closeErr
}

// 读取消息
func (c *serverConn) read() {
	connID := c.ID()
	conn := c.getConn()
	closeCh := c.close

	if conn == nil {
		_ = c.forceCloseIfCurrent(connID, true)
		return
	}

	for {
		select {
		case <-closeCh:
			return
		default:
		}

		msgType, msgData, err := conn.ReadMessage()
		if err != nil {
			if !xerrors.Is(err, net.ErrClosed) {
				if _, ok := err.(*websocket.CloseError); !ok {
					xlog.Logger().Warn("read message failed", zap.Any("id", c.id), zap.Error(err))
				}
			}

			_ = c.forceCloseIfCurrent(connID, true)
			return
		}

		if msgType != websocket.BinaryMessage {
			continue
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
		if len(msgData) == 0 {
			continue
		}

		if !network.TryEnqueueRecv(c.recvQ, msgData) {
			xlog.Logger().Warn("ws receive queue full, close conn", zap.Any("id", c.id))
			_ = c.forceCloseIfCurrent(connID, true)
			return
		}
	}
}

func (c *serverConn) receiveLoop(connID int64, recvQ <-chan []byte) {
	s := c.connMgr.server
	for data := range recvQ {
		if c.ID() != connID {
			return
		}

		if c.State() != network.ConnOpened {
			return
		}

		if s != nil && s.receiveHandler != nil {
			s.receiveHandler(c, data)
		}
	}
}

// 写入消息
// gorilla/websocket 不允许并发写入。
// 所有写入统一进入一个 write goroutine。
// chHighWrite 优先级高，适合心跳、关键控制包。
// chLowWrite 优先级低，适合普通业务消息。
func (c *serverConn) write() {
	connID := c.ID()
	closeCh := c.close
	chHighWrite := c.chHighWrite
	chLowWrite := c.chLowWrite

	for {
		// 优先处理 close，避免强关时卡在写队列。
		select {
		case <-closeCh:
			return
		default:
		}

		// 第一层：非阻塞优先取高优先级消息。
		select {
		case r, ok := <-chHighWrite:
			if !ok {
				c.notifyDone()
				return
			}

			if !c.doWrite(connID, r) {
				return
			}

		default:
			// 第二层：高低队列同时等待，但高优先级仍然有机会先被取到。
			select {
			case <-closeCh:
				return

			case r, ok := <-chHighWrite:
				if !ok {
					c.notifyDone()
					return
				}

				if !c.doWrite(connID, r) {
					return
				}

			case r, ok := <-chLowWrite:
				if !ok {
					c.notifyDone()
					return
				}

				if !c.doWrite(connID, r) {
					return
				}
			}
		}
	}
}

// 执行写入操作
func (c *serverConn) doWrite(connID int64, r chWrite) bool {
	if r.typ == closeSig {
		c.notifyDone()
		return false
	}

	if r.typ != dataPacket {
		return true
	}

	if len(r.msg) == 0 {
		return true
	}

	if c.isClosed() {
		return false
	}

	conn := c.getConn()
	if conn == nil {
		_ = c.forceCloseIfCurrent(connID, true)
		return false
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, r.msg); err != nil {
		if !xerrors.Is(err, net.ErrClosed) {
			if _, ok := err.(*websocket.CloseError); !ok {
				xlog.Logger().Error("write message error", zap.Error(err))
			}
		}

		_ = c.forceCloseIfCurrent(connID, true)
		return false
	}

	return true
}

// 是否已关闭
func (c *serverConn) isClosed() bool {
	return c.State() == network.ConnClosed
}

func (c *serverConn) getConn() *websocket.Conn {
	c.rw.RLock()
	conn := c.conn
	c.rw.RUnlock()
	return conn
}

func (c *serverConn) notifyDone() {
	c.doneOnce.Do(func() {
		close(c.done)
	})
}
