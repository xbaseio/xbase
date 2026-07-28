package xgnet

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/network"
	xnet "github.com/xbaseio/xbase/utils/xnet"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type server struct {
	opts *serverOptions

	engine xnet.Engine

	nextID atomic.Int64

	connMgr *serverConnMgr

	started chan struct{}
	stopped chan struct{}

	startOnce sync.Once
	stopOnce  sync.Once

	startHandler      network.StartHandler
	stopHandler       network.CloseHandler
	connectHandler    network.ConnectHandler
	disconnectHandler network.DisconnectHandler
	receiveHandler    network.ReceiveHandler
}

var _ network.Server = &server{}

func NewServer(opts ...ServerOption) network.Server {
	o := defaultServerOptions()
	for _, opt := range opts {
		opt(o)
	}

	s := &server{
		opts:    o,
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}

	s.connMgr = newServerConnMgr(s)

	return s
}

// Addr 监听地址
func (s *server) Addr() string {
	return s.opts.addr
}

// Start 启动服务器
func (s *server) Start() error {
	if s.opts.certFile != "" || s.opts.keyFile != "" {
		return errors.New("xnet 暂不建议直接启用 TLS，请用 nginx/caddy 做 TLS 终止后转发到 xnet TCP")
	}

	addr := toGnetAddr(s.opts.addr)

	var runErr atomic.Value

	s.startOnce.Do(func() {
		go func() {
			eh := &gnetEventHandler{s: s}

			err := xnet.Run(
				eh,
				addr,
				xnet.WithMulticore(true),
				xnet.WithTCPNoDelay(xnet.TCPNoDelay),
			)

			if err != nil {
				runErr.Store(err)
			}

			close(s.stopped)
		}()
	})

	select {
	case <-s.started:
		if v := runErr.Load(); v != nil {
			return v.(error)
		}
		return nil

	case <-s.stopped:
		if v := runErr.Load(); v != nil {
			return v.(error)
		}
		return xerrors.ErrConnectionClosed

	case <-time.After(3 * time.Second):
		return errors.New("xnet server start timeout")
	}
}

// Stop 关闭服务器
func (s *server) Stop() error {

	var err error

	s.stopOnce.Do(func() {
		if s.connMgr != nil {
			s.connMgr.close()
		}

		if e := s.engine.Validate(); e == nil {
			err = s.engine.Stop(context.Background())
		}
	})

	return err
}

// Protocol 协议
func (s *server) Protocol() string {
	return protocol
}

// OnStart 监听服务器启动
func (s *server) OnStart(handler network.StartHandler) {
	s.startHandler = handler
}

// OnStop 监听服务器关闭
func (s *server) OnStop(handler network.CloseHandler) {
	s.stopHandler = handler
}

// OnConnect 监听连接打开
func (s *server) OnConnect(handler network.ConnectHandler) {
	s.connectHandler = handler
}

// OnDisconnect 监听连接关闭
func (s *server) OnDisconnect(handler network.DisconnectHandler) {
	s.disconnectHandler = handler
}

// OnReceive 监听接收到消息
func (s *server) OnReceive(handler network.ReceiveHandler) {
	s.receiveHandler = handler
}

func toGnetAddr(addr string) string {
	if strings.Contains(addr, "://") {
		return addr
	}

	return "tcp://" + addr
}

type gnetEventHandler struct {
	xnet.BuiltinEventEngine
	s *server
}

func (eh *gnetEventHandler) OnBoot(eng xnet.Engine) xnet.Action {
	eh.s.engine = eng

	select {
	case <-eh.s.started:
	default:
		close(eh.s.started)
	}

	if eh.s.startHandler != nil {
		go eh.s.startHandler()
	}

	return xnet.None
}

func (eh *gnetEventHandler) OnShutdown(eng xnet.Engine) {
	if eh.s.stopHandler != nil {
		eh.s.stopHandler()
	}
}

func (eh *gnetEventHandler) OnOpen(c xnet.Conn) ([]byte, xnet.Action) {
	conn, err := eh.s.connMgr.allocate(c)
	if err != nil {
		xlog.Logger().Error("connection allocate error", zap.Error(err))
		return nil, xnet.Close
	}

	c.SetContext(conn)

	if eh.s.connectHandler != nil {
		eh.s.connectHandler(conn)
	}

	return nil, xnet.None
}

func (eh *gnetEventHandler) OnClose(c xnet.Conn, err error) xnet.Action {
	ctx := c.Context()

	conn, ok := ctx.(*gnetConn)
	if !ok || conn == nil {
		return xnet.None
	}

	conn.markClosed()

	if eh.s.disconnectHandler != nil {
		eh.s.disconnectHandler(conn)
	}

	eh.s.connMgr.recycle(conn.id)

	return xnet.None
}

func (eh *gnetEventHandler) OnTraffic(c xnet.Conn) xnet.Action {
	ctx := c.Context()
	gc, ok := ctx.(*gnetConn)
	if !ok || gc == nil {
		return xnet.Close
	}

	data, err := c.Next(-1)
	if err != nil {
		return xnet.Close
	}

	// xnet 的 Next/Peek 返回的底层 buffer 不能跨 goroutine 使用。
	// 所以这里必须 copy 一份。
	raw := append([]byte(nil), data...)

	if ok = gc.feed(raw); !ok {
		return xnet.Close
	}

	return xnet.None
}
