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

const maxGnetInboundBuffer = 8 * 1024 * 1024

type gnetConnBox struct {
	conn xnet.Conn
}

type gnetConn struct {
	id  int64
	uid atomic.Int64

	attr *attr

	s    *server
	conn atomic.Pointer[gnetConnBox]

	state atomic.Int32

	localAddr  atomic.Value // net.Addr
	remoteAddr atomic.Value // net.Addr

	recvQ chan []byte

	closeOnce sync.Once

	// generation 用来防止连接对象复用后，旧 goroutine 误处理新连接
	generation atomic.Int64

	// inbound 只允许在 gnet event-loop 的 OnTraffic/feed 中访问
	inbound []byte
}

var _ network.Conn = &gnetConn{}

func (c *gnetConn) init(cm *serverConnMgr, id int64, conn xnet.Conn) {
	c.generation.Add(1)
	gen := c.generation.Load()

	c.id = id
	c.uid.Store(0)
	c.attr = &attr{}
	c.s = cm.server
	c.conn.Store(&gnetConnBox{conn: conn})
	c.state.Store(int32(network.ConnOpened))

	c.localAddr = atomic.Value{}
	c.remoteAddr = atomic.Value{}

	if addr := conn.LocalAddr(); addr != nil {
		c.localAddr.Store(addr)
	}

	if addr := conn.RemoteAddr(); addr != nil {
		c.remoteAddr.Store(addr)
	}

	c.recvQ = make(chan []byte, network.DefaultRecvQueueSize)
	c.inbound = make([]byte, 0, 4096)
	c.closeOnce = sync.Once{}

	go c.receiveLoop(gen, c.recvQ, c.s)
}

func (c *gnetConn) reset() {
	c.uid.Store(0)
	c.attr = nil
	c.s = nil
	c.conn.Store(nil)
	c.state.Store(int32(network.ConnClosed))
	c.localAddr = atomic.Value{}
	c.remoteAddr = atomic.Value{}
	c.recvQ = nil
	c.inbound = nil
	c.closeOnce = sync.Once{}
}

func (c *gnetConn) ID() int64 {
	return c.id
}

func (c *gnetConn) UID() int64 {
	return c.uid.Load()
}

func (c *gnetConn) Attr() network.Attr {
	return c.attr
}

func (c *gnetConn) Bind(uid int64) {
	c.uid.Store(uid)
}

func (c *gnetConn) Unbind() {
	c.uid.Store(0)
}

func (c *gnetConn) Send(msg []byte) error {
	return c.Push(msg)
}

func (c *gnetConn) Push(msg []byte) error {
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

	// gnet.Conn.AsyncWrite 是并发安全的，可以在 event-loop 外调用
	if err := conn.AsyncWrite(msg, nil); err != nil {
		return err
	}

	if c.State() != network.ConnOpened {
		return xerrors.ErrConnectionClosed
	}

	return nil
}

func (c *gnetConn) State() network.ConnState {
	return network.ConnState(c.state.Load())
}

func (c *gnetConn) Close(force ...bool) error {
	if !c.state.CompareAndSwap(int32(network.ConnOpened), int32(network.ConnClosed)) {
		return xerrors.ErrConnectionClosed
	}

	conn := c.getConn()
	if conn == nil {
		c.markClosed()
		return xerrors.ErrConnectionClosed
	}

	err := conn.Close()

	// 主动 Close 时也先标记，后续 OnClose 再触发不会重复 close recvQ
	c.markClosed()

	return err
}

func (c *gnetConn) LocalIP() (string, error) {
	addr, err := c.LocalAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

func (c *gnetConn) LocalAddr() (net.Addr, error) {
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

func (c *gnetConn) RemoteIP() (string, error) {
	addr, err := c.RemoteAddr()
	if err != nil {
		return "", err
	}

	return xnet.ExtractIP(addr)
}

func (c *gnetConn) RemoteAddr() (net.Addr, error) {
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

func (c *gnetConn) checkState() error {
	switch c.State() {
	case network.ConnHanged:
		return xerrors.ErrConnectionHanged

	case network.ConnClosed:
		return xerrors.ErrConnectionClosed

	default:
		return nil
	}
}

func (c *gnetConn) getConn() xnet.Conn {
	box := c.conn.Load()
	if box == nil {
		return nil
	}

	return box.conn
}

func (c *gnetConn) markClosed() {
	c.closeOnce.Do(func() {
		c.state.Store(int32(network.ConnClosed))
		c.conn.Store(nil)

		if c.recvQ != nil {
			close(c.recvQ)
		}
	})
}

func (c *gnetConn) receiveLoop(gen int64, recvQ <-chan []byte, s *server) {
	for data := range recvQ {
		// 对象已经被复用，旧 goroutine 直接退出
		if c.generation.Load() != gen {
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

// feed 处理 TCP 粘包/半包。
// 注意：这个方法只应该在 gnet event-loop 中调用。
func (c *gnetConn) feed(raw []byte) bool {
	if len(raw) == 0 {
		return true
	}

	if c.State() != network.ConnOpened {
		return false
	}

	c.inbound = append(c.inbound, raw...)

	if len(c.inbound) > maxGnetInboundBuffer {
		return false
	}

	for len(c.inbound) > 0 {
		reader := bytes.NewReader(c.inbound)

		data, err := packet.ReadMessage(reader)
		if err != nil {
			// 可能只是半包，等下次 OnTraffic 继续 append 再解析。
			return true
		}

		consumed := len(c.inbound) - reader.Len()
		if consumed <= 0 {
			return false
		}

		c.inbound = c.inbound[consumed:]

		// data 需要复制，避免底层缓冲复用带来的问题
		if !network.TryEnqueueRecv(c.recvQ, data) {
			return false
		}
	}

	return true
}
