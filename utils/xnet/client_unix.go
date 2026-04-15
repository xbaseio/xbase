//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	"context"
	"errors"
	"net"
	"strconv"
	"syscall"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xbuffer/xring"
	"github.com/xbaseio/xbase/xerrors"

	"github.com/xbaseio/xbase/utils/xmath"
	"github.com/xbaseio/xbase/utils/xnetpoll"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/utils/xsocket"
)

//
// ========================= Client =========================
//

// Client 表示 xnet 客户端，负责连接管理和事件循环调度
type Client struct {
	opts *Options // 配置项
	eng  *engine  // 核心引擎
}

//
// ========================= 初始化 =========================
//

// NewClient 创建一个新的 Client 实例
func NewClient(eh EventHandler, opts ...Option) (*Client, error) {
	options := loadOptions(opts...)

	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)

	eng := &engine{
		listeners:    make(map[int]*listener),
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eh,
		eventLoops:   new(leastConnectionsLoadBalancer),
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}

	normalizeOptions(options)

	return &Client{
		opts: options,
		eng:  eng,
	}, nil
}

// normalizeOptions 统一处理配置参数（buffer / ET 模式）
func normalizeOptions(opts *Options) {
	// ET 模式处理
	if opts.EdgeTriggeredIOChunk > 0 {
		opts.EdgeTriggeredIO = true
		opts.EdgeTriggeredIOChunk = xmath.CeilToPowerOfTwo(opts.EdgeTriggeredIOChunk)
	} else if opts.EdgeTriggeredIO {
		opts.EdgeTriggeredIOChunk = 1 << 20 // 默认 1MB
	}

	// Read buffer
	opts.ReadBufferCap = normalizeBufferCap(opts.ReadBufferCap)

	// Write buffer
	opts.WriteBufferCap = normalizeBufferCap(opts.WriteBufferCap)
}

// normalizeBufferCap 统一 buffer 大小策略
func normalizeBufferCap(size int) int {
	switch {
	case size <= 0:
		return MaxStreamBufferCap
	case size <= xring.DefaultBufferSize:
		return xring.DefaultBufferSize
	default:
		return xmath.CeilToPowerOfTwo(size)
	}
}

//
// ========================= 生命周期 =========================
//

// Start 启动客户端事件循环
func (cli *Client) Start() error {
	numEventLoop := determineEventLoops(cli.opts)
	log.Infof("Starting xnet client with %d event loops", numEventLoop)

	cli.eng.eventHandler.OnBoot(Engine{cli.eng})

	var tickerLoop *eventloop

	// 初始化 event loops
	for i := 0; i < numEventLoop; i++ {
		p, err := xnetpoll.OpenPoller()
		if err != nil {
			cli.eng.closeEventLoops()
			return err
		}

		el := &eventloop{
			listeners:    cli.eng.listeners,
			engine:       cli.eng,
			poller:       p,
			buffer:       make([]byte, cli.opts.ReadBufferCap),
			eventHandler: cli.eng.eventHandler,
		}

		el.connections.init()
		cli.eng.eventLoops.register(el)

		// ticker 只在第一个 loop 上执行
		if cli.opts.Ticker && el.idx == 0 {
			tickerLoop = el
		}
	}

	// 启动所有 event loop
	cli.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		cli.eng.concurrency.Go(el.run)
		return true
	})

	// 启动 ticker
	if tickerLoop != nil {
		ctx := cli.eng.concurrency.ctx
		cli.eng.concurrency.Go(func() error {
			tickerLoop.ticker(ctx)
			return nil
		})
	}

	log.Debugf("default log level is %s", log.LogLevel())
	return nil
}

// Stop 停止客户端并关闭所有事件循环
func (cli *Client) Stop() error {
	cli.eng.shutdown(nil)

	cli.eng.eventHandler.OnShutdown(Engine{cli.eng})

	// 通知所有 event loop 退出
	cli.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		log.Error(el.poller.Trigger(
			xqueue.HighPriority,
			func(_ any) error { return xerrors.ErrEngineShutdown },
			nil,
		))
		return true
	})

	// 等待所有 goroutine 退出
	err := cli.eng.concurrency.Wait()

	cli.eng.closeEventLoops()
	cli.eng.inShutdown.Store(true)

	return err
}

//
// ========================= 连接操作 =========================
//

// Dial 等价于 net.Dial
func (cli *Client) Dial(network, address string) (Conn, error) {
	return cli.DialContext(network, address, nil)
}

// DialContext 带上下文的连接创建
func (cli *Client) DialContext(network, address string, ctx any) (Conn, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return cli.EnrollContext(c, ctx)
}

// Enroll 将 net.Conn 转换为 xnet.Conn
func (cli *Client) Enroll(c net.Conn) (Conn, error) {
	return cli.EnrollContext(c, nil)
}

// EnrollContext 注册连接到事件循环（核心路径）
func (cli *Client) EnrollContext(c net.Conn, ctx any) (Conn, error) {
	defer c.Close() // 原始连接由 fd 接管

	// 获取底层 fd
	sc, ok := c.(syscall.Conn)
	if !ok {
		return nil, errors.New("net.Conn 转 syscall.Conn 失败")
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return nil, errors.New("获取 RawConn 失败")
	}

	var dupFD int
	e := rc.Control(func(fd uintptr) {
		dupFD, err = xsocket.Dup(int(fd))
	})
	if err != nil {
		return nil, err
	}
	if e != nil {
		return nil, e
	}

	// socket buffer 设置
	if cli.opts.SocketSendBuffer > 0 {
		if err = xsocket.SetSendBuffer(dupFD, cli.opts.SocketSendBuffer); err != nil {
			return nil, err
		}
	}
	if cli.opts.SocketRecvBuffer > 0 {
		if err = xsocket.SetRecvBuffer(dupFD, cli.opts.SocketRecvBuffer); err != nil {
			return nil, err
		}
	}

	el := cli.eng.eventLoops.next(nil)

	var (
		sockAddr unix.Sockaddr
		gc       *conn
	)

	// 根据连接类型构建不同连接对象
	switch conn := c.(type) {

	case *net.UnixConn:
		sockAddr, _, _, err = xsocket.GetUnixSockAddr(
			conn.RemoteAddr().Network(),
			conn.RemoteAddr().String(),
		)
		if err != nil {
			return nil, err
		}

		ua := conn.LocalAddr().(*net.UnixAddr)
		ua.Name = conn.RemoteAddr().String() + "." + strconv.Itoa(dupFD)

		gc = newStreamConn("unix", dupFD, el, sockAddr, conn.LocalAddr(), conn.RemoteAddr())

	case *net.TCPConn:
		if cli.opts.TCPNoDelay == TCPNoDelay {
			if err = xsocket.SetNoDelay(dupFD, 1); err != nil {
				return nil, err
			}
		}

		if cli.opts.TCPKeepAlive > 0 {
			if err = setKeepAlive(
				dupFD,
				true,
				cli.opts.TCPKeepAlive,
				cli.opts.TCPKeepInterval,
				cli.opts.TCPKeepCount,
			); err != nil {
				return nil, err
			}
		}

		sockAddr, _, _, _, err = xsocket.GetTCPSockAddr(
			conn.RemoteAddr().Network(),
			conn.RemoteAddr().String(),
		)
		if err != nil {
			return nil, err
		}

		gc = newStreamConn("tcp", dupFD, el, sockAddr, conn.LocalAddr(), conn.RemoteAddr())

	case *net.UDPConn:
		sockAddr, _, _, _, err = xsocket.GetUDPSockAddr(
			conn.RemoteAddr().Network(),
			conn.RemoteAddr().String(),
		)
		if err != nil {
			return nil, err
		}

		gc = newUDPConn(dupFD, el, conn.LocalAddr(), sockAddr, true)

	default:
		return nil, xerrors.ErrUnsupportedProtocol
	}

	gc.ctx = ctx

	// 注册到 poller（必须在 eventloop 内执行）
	connOpened := make(chan struct{})

	ccb := &connWithCallback{
		c: gc,
		cb: func() {
			close(connOpened)
		},
	}

	if err = el.poller.Trigger(xqueue.HighPriority, el.register, ccb); err != nil {
		gc.Close()
		return nil, err
	}

	<-connOpened

	return gc, nil
}
