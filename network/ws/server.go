package ws

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/utils/xcall"
)

type UpgradeHandler func(w http.ResponseWriter, r *http.Request) (allowed bool)

type Server interface {
	network.Server

	// OnUpgrade 监听HTTP请求升级
	OnUpgrade(handler UpgradeHandler)
}

type server struct {
	opts     *serverOptions
	listener net.Listener

	httpServer *http.Server

	connMgr *serverConnMgr

	startHandler      network.StartHandler
	stopHandler       network.CloseHandler
	connectHandler    network.ConnectHandler
	disconnectHandler network.DisconnectHandler
	receiveHandler    network.ReceiveHandler
	upgradeHandler    UpgradeHandler
}

var _ Server = &server{}

func NewServer(opts ...ServerOption) Server {
	o := defaultServerOptions()
	for _, opt := range opts {
		opt(o)
	}

	s := &server{
		opts: o,
	}

	s.connMgr = newConnMgr(s)

	return s
}

// Addr 监听地址
func (s *server) Addr() string {
	return s.opts.addr
}

// Protocol 协议
func (s *server) Protocol() string {
	return protocol
}

// Start 启动服务器
func (s *server) Start() error {
	if err := s.init(); err != nil {
		return err
	}

	if s.startHandler != nil {
		s.startHandler()
	}

	xcall.Go(s.serve)

	return nil
}

// Stop 关闭服务器
func (s *server) Stop() error {
	var retErr error

	// 先关闭业务连接，websocket 升级后属于 hijack 连接，
	// http.Server.Shutdown 不一定能管理到这些连接。
	if s.connMgr != nil {
		s.connMgr.close()
	}

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := s.httpServer.Shutdown(ctx)
		cancel()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			retErr = err

			// Shutdown 超时兜底 Close
			_ = s.httpServer.Close()
		}
	} else if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			retErr = err
		}
	}

	if s.stopHandler != nil {
		s.stopHandler()
	}

	return retErr
}

// 初始化服务器
func (s *server) init() error {
	addr := normalizeListenAddr(s.opts.addr)
	if addr == "" {
		return errors.New("websocket listen addr is empty")
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = ln
	s.opts.addr = ln.Addr().String()

	return nil
}

// 启动服务器
func (s *server) serve() {
	mux := http.NewServeMux()

	upgrader := websocket.Upgrader{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: false,
		CheckOrigin:       s.opts.checkOrigin,
	}

	path := s.opts.path
	if path == "" {
		path = "/"
	}

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		s.handleUpgrade(w, r, &upgrader)
	})

	s.httpServer = &http.Server{
		Handler: mux,

		// 防止慢请求拖死 HTTP Upgrade 阶段
		ReadHeaderTimeout: 5 * time.Second,

		// Upgrade 之前的普通 HTTP 读超时
		ReadTimeout: 10 * time.Second,

		// Upgrade 响应阶段写超时
		WriteTimeout: 10 * time.Second,

		// KeepAlive 空闲超时
		IdleTimeout: 60 * time.Second,
	}

	var err error
	if s.opts.certFile != "" && s.opts.keyFile != "" {
		err = s.httpServer.ServeTLS(s.listener, s.opts.certFile, s.opts.keyFile)
	} else {
		err = s.httpServer.Serve(s.listener)
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("websocket server shutdown, addr=%s, err=%v", s.opts.addr, err)
	}
}

// 处理 HTTP -> WebSocket 升级
func (s *server) handleUpgrade(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if s.upgradeHandler != nil && !s.upgradeHandler(w, r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("websocket upgrade error, remote=%s, path=%s, err=%v", r.RemoteAddr, r.URL.Path, err)
		return
	}

	if err = s.connMgr.allocate(conn); err != nil {
		log.Errorf("connection allocate error, remote=%s, err=%v", r.RemoteAddr, err)
		_ = conn.Close()
		return
	}
}

// 兼容传入 "3653" / ":3653" / "0.0.0.0:3653"
func normalizeListenAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	if _, err := strconv.Atoi(addr); err == nil {
		return ":" + addr
	}

	return addr
}

// OnStart 监听服务器启动
func (s *server) OnStart(handler network.StartHandler) {
	s.startHandler = handler
}

// OnStop 监听服务器关闭
func (s *server) OnStop(handler network.CloseHandler) {
	s.stopHandler = handler
}

// OnUpgrade 监听HTTP请求升级
func (s *server) OnUpgrade(handler UpgradeHandler) {
	s.upgradeHandler = handler
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
