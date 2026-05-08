package ws

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/utils/xtime"
	"github.com/xbaseio/xbase/xerrors"
)

type clientConn struct {
	rw sync.RWMutex // conn 保护锁

	// 发送锁：
	// 1. 防止 Send/Push 和关闭写队列并发导致 panic
	// 2. 保证 graceClose 的 closeSig 排在已经进入 Send/Push 的消息之后
	sendMu sync.Mutex

	id                int64           // 连接ID
	uid               atomic.Int64    // 用户ID
	attr              *attr           // 连接属性
	conn              *websocket.Conn // WS源连接
	state             atomic.Int32    // 连接状态
	client            *client         // 客户端
	chLowWrite        chan chWrite    // 低级队列
	chHighWrite       chan chWrite    // 优先队列
	lastHeartbeatTime atomic.Int64    // 上次心跳时间
	done              chan struct{}   // 写入完成信号，使用 close 通知
	close             chan struct{}   // 关闭信号

	doneOnce  sync.Once
	closeOnce sync.Once
}

var _ network.Conn = &clientConn{}

func newClientConn(id int64, conn *websocket.Conn, client *client) network.Conn {
	c := &clientConn{
		id:          id,
		attr:        &attr{},
		conn:        conn,
		client:      client,
		chLowWrite:  make(chan chWrite, 4096),
		chHighWrite: make(chan chWrite, 1024),
		done:        make(chan struct{}),
		close:       make(chan struct{}),
	}

	c.state.Store(int32(network.ConnOpened))
	c.lastHeartbeatTime.Store(xtime.Now().UnixNano())

	xcall.Go(c.read)
	xcall.Go(c.write)

	if c.client.connectHandler != nil {
		c.client.connectHandler(c)
	}

	return c
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
// gorilla/websocket 不允许并发写，所以 Send 也进入高优先级写队列。
func (c *clientConn) Send(msg []byte) error {
	return c.enqueueWrite(c.chHighWrite, msg)
}

// Push 发送消息（异步）
func (c *clientConn) Push(msg []byte) error {
	return c.enqueueWrite(c.chLowWrite, msg)
}

func (c *clientConn) enqueueWrite(ch chan chWrite, msg []byte) error {
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
	}
}

// State 获取连接状态
func (c *clientConn) State() network.ConnState {
	return network.ConnState(c.state.Load())
}

// Close 关闭连接（主动关闭）
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

	// ConnHanged 后新的 Send/Push 会被拒绝。
	// 这里拿 sendMu，等待已经进入 Send/Push 的请求完成。
	c.sendMu.Lock()

	if c.State() == network.ConnClosed {
		c.sendMu.Unlock()
		return nil
	}

	if c.getConn() == nil {
		c.sendMu.Unlock()
		return c.finishGraceClose()
	}

	select {
	case c.chLowWrite <- chWrite{typ: closeSig}:
		c.sendMu.Unlock()

	case <-c.close:
		c.sendMu.Unlock()
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
		// 先关闭 close，让 Send/Push/graceClose/read/write 中的 select 尽快退出。
		close(c.close)

		// done 使用 close 通知，避免无缓冲 channel 发送阻塞。
		c.notifyDone()

		// 等待正在 Send/Push 的 goroutine 退出，然后再关闭写队列。
		// 防止 close(channel) 和 channel <- value 并发导致 panic。
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

		msgType, msgData, err := conn.ReadMessage()
		if err != nil {
			if !xerrors.Is(err, net.ErrClosed) {
				if _, ok := err.(*websocket.CloseError); !ok {
					log.Warnf("read message failed: %v", err)
				}
			}

			_ = c.forceClose()
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

		if c.client.receiveHandler != nil {
			c.client.receiveHandler(c, msgData)
		}
	}
}

// 写入消息
// gorilla/websocket 不允许并发写入。
// 所有写入统一进入一个 write goroutine。
// chHighWrite 优先级高，适合心跳、关键控制包。
// chLowWrite 优先级低，适合普通业务消息。
func (c *clientConn) write() {
	closeCh := c.close
	chHighWrite := c.chHighWrite
	chLowWrite := c.chLowWrite

	for {
		// 优先处理关闭信号，避免强关时卡在写队列。
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

			if !c.doWrite(r) {
				return
			}

		default:
			// 第二层：高低队列同时等待。
			select {
			case <-closeCh:
				return

			case r, ok := <-chHighWrite:
				if !ok {
					c.notifyDone()
					return
				}

				if !c.doWrite(r) {
					return
				}

			case r, ok := <-chLowWrite:
				if !ok {
					c.notifyDone()
					return
				}

				if !c.doWrite(r) {
					return
				}
			}
		}
	}
}

// 执行写入操作
func (c *clientConn) doWrite(r chWrite) bool {
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
		_ = c.forceClose()
		return false
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, r.msg); err != nil {
		if !xerrors.Is(err, net.ErrClosed) {
			if _, ok := err.(*websocket.CloseError); !ok {
				log.Errorf("write message error: %v", err)
			}
		}

		_ = c.forceClose()
		return false
	}

	return true
}

// 是否已关闭
func (c *clientConn) isClosed() bool {
	return c.State() == network.ConnClosed
}

func (c *clientConn) getConn() *websocket.Conn {
	c.rw.RLock()
	conn := c.conn
	c.rw.RUnlock()
	return conn
}

func (c *clientConn) notifyDone() {
	c.doneOnce.Do(func() {
		close(c.done)
	})
}
