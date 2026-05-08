package xgnet

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/xerrors"
)

const maxGnetClientInboundBuffer = 8 * 1024 * 1024

type gnetClientConnBox struct {
	conn xnet.Conn
}

type gnetClientConn struct {
	id  int64
	uid atomic.Int64

	attr *attr

	client *client
	conn   atomic.Pointer[gnetClientConnBox]
	state  atomic.Int32

	localAddr  atomic.Value // net.Addr
	remoteAddr atomic.Value // net.Addr

	recvQ chan []byte

	closeOnce sync.Once

	// inbound 只在 xnet event-loop 的 OnTraffic 中访问，不需要锁。
	inbound []byte
}

var _ network.Conn = &gnetClientConn{}

func newGnetClientConn(client *client, id int64) *gnetClientConn {
	c := &gnetClientConn{
		id:      id,
		attr:    &attr{},
		client:  client,
		recvQ:   make(chan []byte, 1024),
		inbound: make([]byte, 0, 4096),
	}

	c.state.Store(int32(network.ConnOpened))

	go c.receiveLoop()

	return c
}

func (c *gnetClientConn) ID() int64 {
	return c.id
}

func (c *gnetClientConn) UID() int64 {
	return c.uid.Load()
}

func (c *gnetClientConn) Attr() network.Attr {
	return c.attr
}

func (c *gnetClientConn) Bind(uid int64) {
	c.uid.Store(uid)
}

func (c *gnetClientConn) Unbind() {
	c.uid.Store(0)
}

// Send 发送消息。
// xnet 版 Send/Push 都走 AsyncWrite。
// AsyncWrite 是并发安全的。
func (c *gnetClientConn) Send(msg []byte) error {
	return c.Push(msg)
}

// Push 发送消息。
func (c *gnetClientConn) Push(msg []byte) error {
	if len(msg) == 0 {
		return nil
	}

	if c.State() != network.ConnOpened {
		return c.checkState()
	}

	conn := c.getConn()
	if conn == nil {
		return xerrors.ErrConnectionClosed
	}

	if err := conn.AsyncWrite(msg, nil); err != nil {
		return err
	}

	if c.State() != network.ConnOpened {
		return xerrors.ErrConnectionClosed
	}

	return nil
}

func (c *gnetClientConn) State() network.ConnState {
	return network.ConnState(c.state.Load())
}

func (c *gnetClientConn) Close(force ...bool) error {
	if c.state.Swap(int32(network.ConnClosed)) == int32(network.ConnClosed) {
		return xerrors.ErrConnectionClosed
	}

	conn := c.getConn()
	if conn == nil {
		c.markClosed()
		return xerrors.ErrConnectionClosed
	}

	err := conn.Close()

	// 主动关闭时也直接标记，OnClose 后续触发时不会重复处理。
	c.markClosed()

	return err
}

func (c *gnetClientConn) LocalIP() (string, error) {
	addr, err := c.LocalAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

func (c *gnetClientConn) LocalAddr() (net.Addr, error) {
	v := c.localAddr.Load()
	if v == nil {
		return nil, xerrors.ErrConnectionClosed
	}

	addr, ok := v.(net.Addr)
	if !ok || addr == nil {
		return nil, xerrors.ErrConnectionClosed
	}

	return addr, nil
}

func (c *gnetClientConn) RemoteIP() (string, error) {
	addr, err := c.RemoteAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

func (c *gnetClientConn) RemoteAddr() (net.Addr, error) {
	v := c.remoteAddr.Load()
	if v == nil {
		return nil, xerrors.ErrConnectionClosed
	}

	addr, ok := v.(net.Addr)
	if !ok || addr == nil {
		return nil, xerrors.ErrConnectionClosed
	}

	return addr, nil
}

func (c *gnetClientConn) checkState() error {
	switch c.State() {
	case network.ConnHanged:
		return xerrors.ErrConnectionHanged

	case network.ConnClosed:
		return xerrors.ErrConnectionClosed

	default:
		return nil
	}
}

func (c *gnetClientConn) setConn(conn xnet.Conn) {
	if conn == nil {
		return
	}

	c.conn.Store(&gnetClientConnBox{conn: conn})
}

func (c *gnetClientConn) attach(conn xnet.Conn) {
	if conn == nil {
		return
	}

	c.setConn(conn)

	if addr := conn.LocalAddr(); addr != nil {
		c.localAddr.Store(addr)
	}

	if addr := conn.RemoteAddr(); addr != nil {
		c.remoteAddr.Store(addr)
	}
}

func (c *gnetClientConn) getConn() xnet.Conn {
	box := c.conn.Load()
	if box == nil {
		return nil
	}

	return box.conn
}

func (c *gnetClientConn) markClosed() {
	c.closeOnce.Do(func() {
		c.state.Store(int32(network.ConnClosed))
		c.conn.Store(nil)
		close(c.recvQ)
	})
}

func (c *gnetClientConn) receiveLoop() {
	for data := range c.recvQ {
		if c.client.receiveHandler != nil {
			c.client.receiveHandler(c, data)
		}
	}
}

// feed 处理 TCP 粘包/半包。
// 这里用 packet.ReadMessage(bytes.Reader) 复用你现有 packet 协议。
// 后面如果要更高性能，建议在 packet 包里加 TryReadFrame([]byte) 零拷贝解析。
func (c *gnetClientConn) feed(raw []byte) bool {
	if len(raw) == 0 {
		return true
	}

	c.inbound = append(c.inbound, raw...)

	if len(c.inbound) > maxGnetClientInboundBuffer {
		return false
	}

	for len(c.inbound) > 0 {
		reader := bytes.NewReader(c.inbound)

		data, err := packet.ReadMessage(reader)
		if err != nil {
			// 可能只是半包，等下次 OnTraffic 再继续解析。
			return true
		}

		consumed := len(c.inbound) - reader.Len()
		if consumed <= 0 {
			return false
		}

		c.inbound = c.inbound[consumed:]

		frame := append([]byte(nil), data...)

		select {
		case c.recvQ <- frame:
		default:
			// 业务处理太慢，避免 event-loop 堆积。
			return false
		}
	}

	return true
}
