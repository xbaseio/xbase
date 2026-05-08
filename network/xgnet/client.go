package xgnet

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/utils/xnet"
)

type client struct {
	opts *clientOptions // 配置
	id   atomic.Int64   // 连接ID

	gcli      *xnet.Client
	startOnce sync.Once
	startErr  error

	connectHandler    network.ConnectHandler    // 连接打开hook函数
	disconnectHandler network.DisconnectHandler // 连接关闭hook函数
	receiveHandler    network.ReceiveHandler    // 接收消息hook函数
}

var _ network.Client = &client{}

func NewClient(opts ...ClientOption) network.Client {
	o := defaultClientOptions()
	for _, opt := range opts {
		opt(o)
	}

	return &client{opts: o}
}

// Dial 拨号连接
func (c *client) Dial(addr ...string) (network.Conn, error) {
	if c.opts.caFile != "" {
		return nil, errors.New("xtcp xnet client does not support TLS directly; use nginx/caddy TLS termination or keep the net/tls client")
	}

	if err := c.startGnetClient(); err != nil {
		return nil, err
	}

	address := c.opts.addr
	if len(addr) > 0 && addr[0] != "" {
		address = addr[0]
	}

	// 保持和 net.DialTimeout 类似的超时语义。
	timeout := c.opts.timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	id := c.id.Add(1)
	conn := newGnetClientConn(c, id)

	done := make(chan struct{})
	var (
		gc  xnet.Conn
		err error
	)

	go func() {
		gc, err = c.gcli.DialContext("tcp", address, conn)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			conn.markClosed()
			return nil, err
		}

		conn.setConn(gc)
		return conn, nil

	case <-time.After(timeout):
		conn.markClosed()
		return nil, net.ErrClosed
	}
}

func (c *client) startGnetClient() error {
	c.startOnce.Do(func() {
		handler := &gnetClientEventHandler{
			client:  c,
			started: make(chan struct{}),
		}

		gcli, err := xnet.NewClient(handler)
		if err != nil {
			c.startErr = err
			return
		}

		c.gcli = gcli

		done := make(chan error, 1)

		go func() {
			done <- gcli.Start()
		}()

		timeout := c.opts.timeout
		if timeout <= 0 {
			timeout = 3 * time.Second
		}

		select {
		case <-handler.started:
			// started

		case err = <-done:
			c.startErr = err

		case <-time.After(timeout):
			c.startErr = errors.New("xnet client start timeout")
		}
	})

	return c.startErr
}

// Stop 停止 xnet client。
// network.Client 接口里如果没有 Stop，也不影响，外部需要时可以类型断言调用。
func (c *client) Stop() error {
	if c.gcli == nil {
		return nil
	}

	return c.gcli.Stop()
}

// Protocol 协议
func (c *client) Protocol() string {
	return protocol
}

// OnConnect 监听连接打开
func (c *client) OnConnect(handler network.ConnectHandler) {
	c.connectHandler = handler
}

// OnDisconnect 监听连接关闭
func (c *client) OnDisconnect(handler network.DisconnectHandler) {
	c.disconnectHandler = handler
}

// OnReceive 监听接收到消息
func (c *client) OnReceive(handler network.ReceiveHandler) {
	c.receiveHandler = handler
}

type gnetClientEventHandler struct {
	xnet.BuiltinEventEngine

	client  *client
	started chan struct{}

	startOnce sync.Once
}

func (h *gnetClientEventHandler) OnBoot(_ xnet.Engine) xnet.Action {
	h.startOnce.Do(func() {
		close(h.started)
	})

	return xnet.None
}

func (h *gnetClientEventHandler) OnOpen(c xnet.Conn) ([]byte, xnet.Action) {
	ctx := c.Context()

	conn, ok := ctx.(*gnetClientConn)
	if !ok || conn == nil {
		return nil, xnet.Close
	}

	conn.attach(c)

	if h.client.connectHandler != nil {
		go h.client.connectHandler(conn)
	}

	return nil, xnet.None
}

func (h *gnetClientEventHandler) OnClose(c xnet.Conn, err error) xnet.Action {
	ctx := c.Context()

	conn, ok := ctx.(*gnetClientConn)
	if !ok || conn == nil {
		return xnet.None
	}

	conn.markClosed()

	if h.client.disconnectHandler != nil {
		go h.client.disconnectHandler(conn)
	}

	return xnet.None
}

func (h *gnetClientEventHandler) OnTraffic(c xnet.Conn) xnet.Action {
	ctx := c.Context()

	conn, ok := ctx.(*gnetClientConn)
	if !ok || conn == nil {
		return xnet.Close
	}

	n := c.InboundBuffered()
	if n <= 0 {
		return xnet.None
	}

	data, err := c.Next(n)
	if err != nil {
		return xnet.Close
	}

	// xnet 的 Next/Peek 返回的 buffer 不能跨 goroutine 使用，所以必须拷贝。
	raw := append([]byte(nil), data...)

	if ok = conn.feed(raw); !ok {
		return xnet.Close
	}

	return xnet.None
}
