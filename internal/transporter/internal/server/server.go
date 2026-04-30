package server

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/core/endpoint"
	xnet "github.com/xbaseio/xbase/core/net"
	"github.com/xbaseio/xbase/internal/transporter/internal/codes"
	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
	"github.com/xbaseio/xbase/internal/transporter/internal/route"
	"github.com/xbaseio/xbase/log"
)

const scheme = "drpc"

type RouteHandler func(conn *Conn, data []byte) error

type Server struct {
	listener    net.Listener           // 监听器
	listenAddr  string                 // 监听地址
	exposeAddr  string                 // 暴露地址
	endpoint    *endpoint.Endpoint     // 暴露端点
	handlers    map[uint8]RouteHandler // 路由处理器
	rw          sync.RWMutex           // 锁
	connections map[net.Conn]*Conn     // 连接
	closed      int32                  // 是否关闭
}

func NewServer(opts *Options) (*Server, error) {
	listenAddr, exposeAddr, err := xnet.ParseAddr(opts.Addr, opts.Expose)
	if err != nil {
		return nil, err
	}

	s := &Server{}
	s.listenAddr = listenAddr
	s.exposeAddr = exposeAddr
	s.endpoint = endpoint.NewEndpoint(scheme, exposeAddr, false)
	s.connections = make(map[net.Conn]*Conn)
	s.handlers = make(map[uint8]RouteHandler)

	// 协议层路由：握手
	s.handlers[route.Handshake] = s.handshake

	return s, nil
}

// Scheme 协议
func (s *Server) Scheme() string {
	return scheme
}

// ListenAddr 监听地址
func (s *Server) ListenAddr() string {
	return s.listenAddr
}

// ExposeAddr 暴露地址
func (s *Server) ExposeAddr() string {
	return s.exposeAddr
}

// Endpoint 暴露端点
func (s *Server) Endpoint() *endpoint.Endpoint {
	return s.endpoint
}

// Start 启动服务器
func (s *Server) Start() error {
	if atomic.LoadInt32(&s.closed) == 1 {
		return net.ErrClosed
	}

	addr, err := net.ResolveTCPAddr("tcp", s.listenAddr)
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP(addr.Network(), addr)
	if err != nil {
		return err
	}

	s.rw.Lock()
	if atomic.LoadInt32(&s.closed) == 1 {
		s.rw.Unlock()
		_ = ln.Close()
		return net.ErrClosed
	}
	s.listener = ln
	s.rw.Unlock()

	var tempDelay time.Duration

	for {
		rawConn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			if e, ok := err.(net.Error); ok && e.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > time.Second {
					tempDelay = time.Second
				}

				log.Warnf("tcp accept connect error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			log.Warnf("tcp accept connect error: %v", err)
			return err
		}

		tempDelay = 0

		s.allocate(rawConn)
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	var ln net.Listener
	var conns []*Conn

	s.rw.Lock()

	ln = s.listener
	s.listener = nil

	for _, conn := range s.connections {
		conns = append(conns, conn)
	}

	// 不要设置成 nil，避免并发 allocate 时 panic
	s.connections = make(map[net.Conn]*Conn)

	s.rw.Unlock()

	var err error
	if ln != nil {
		err = ln.Close()
	}

	// 不要在 s.rw 锁内 close，避免 conn.close -> recycle 再抢锁导致死锁
	for _, conn := range conns {
		_ = conn.close(false)
	}

	return err
}

// RegisterHandler 注册处理器
func (s *Server) RegisterHandler(routeID uint8, handler RouteHandler) {
	s.rw.Lock()
	s.handlers[routeID] = handler
	s.rw.Unlock()
}

// getHandler 获取路由处理器
func (s *Server) getHandler(routeID uint8) RouteHandler {
	s.rw.RLock()
	handler := s.handlers[routeID]
	s.rw.RUnlock()
	return handler
}

// 分配连接
func (s *Server) allocate(rawConn net.Conn) {
	conn := newConn(s, rawConn)

	s.rw.Lock()

	if atomic.LoadInt32(&s.closed) == 1 {
		s.rw.Unlock()
		_ = conn.close(false)
		return
	}

	s.connections[rawConn] = conn

	s.rw.Unlock()

	conn.start()
}

// 回收连接
func (s *Server) recycle(rawConn net.Conn) {
	s.rw.Lock()
	delete(s.connections, rawConn)
	s.rw.Unlock()
}

// 处理握手
func (s *Server) handshake(conn *Conn, data []byte) error {
	seq, insKind, insID, err := protocol.DecodeHandshakeReq(data)
	if err != nil {
		return err
	}

	conn.InsID = insID
	conn.InsKind = insKind

	return conn.Send(protocol.EncodeHandshakeRes(seq, codes.ErrorToCode(nil)))
}
