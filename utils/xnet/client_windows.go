package xnet

import (
	"context"
	"net"

	"golang.org/x/sync/errgroup"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xpool/xgoroutine"
	"github.com/xbaseio/xbase/xerrors"
)

// Client 表示 xnet 客户端。
type Client struct {
	opts *Options
	eng  *engine
}

// NewClient 创建一个新的客户端实例。
func NewClient(eh EventHandler, opts ...Option) (cli *Client, err error) {
	options := loadOptions(opts...)
	cli = &Client{opts: options}

	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)

	eng := engine{
		listeners:    []*listener{},
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eh,
		eventLoops:   new(leastConnectionsLoadBalancer),
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}
	cli.eng = &eng

	return
}

// Start 启动客户端事件循环。
func (cli *Client) Start() error {
	numEventLoop := determineEventLoops(cli.opts)
	log.Infof("Starting xnet client with %d event loops", numEventLoop)

	cli.eng.eventHandler.OnBoot(Engine{cli.eng})

	var tickerLoop *eventloop

	for i := 0; i < numEventLoop; i++ {
		el := eventloop{
			ch:           make(chan any, 1024),
			eng:          cli.eng,
			connections:  make(map[*conn]struct{}),
			eventHandler: cli.eng.eventHandler,
		}

		cli.eng.eventLoops.register(&el)
		cli.eng.concurrency.Go(el.run)

		if cli.opts.Ticker && el.idx == 0 {
			tickerLoop = &el
		}
	}

	// 启动 ticker。
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

// Stop 停止客户端事件循环。
func (cli *Client) Stop() error {
	cli.eng.shutdown(nil)

	cli.eng.eventHandler.OnShutdown(Engine{cli.eng})

	// 通知所有事件循环退出。
	cli.eng.closeEventLoops()

	// 等待所有事件循环退出。
	err := cli.eng.concurrency.Wait()

	// 将引擎置为已关闭状态。
	cli.eng.inShutdown.Store(true)

	return err
}

// Dial 等价于 net.Dial。
func (cli *Client) Dial(network, addr string) (Conn, error) {
	return cli.DialContext(network, addr, nil)
}

// DialContext 建立连接，并绑定一个自定义上下文。
func (cli *Client) DialContext(network, addr string, ctx any) (Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return cli.EnrollContext(c, ctx)
}

// Enroll 将 net.Conn 注册到 xnet 客户端中。
func (cli *Client) Enroll(nc net.Conn) (gc Conn, err error) {
	return cli.EnrollContext(nc, nil)
}

// EnrollContext 将 net.Conn 注册到事件循环，并绑定上下文。
func (cli *Client) EnrollContext(nc net.Conn, ctx any) (gc Conn, err error) {
	el := cli.eng.eventLoops.next(nil)
	connOpened := make(chan struct{})

	switch conn := nc.(type) {
	case *net.TCPConn:
		if cli.opts.TCPNoDelay == TCPNoDelay {
			if err = conn.SetNoDelay(true); err != nil {
				return
			}
		}

		c := newStreamConn(el, nc, ctx)

		if opts := cli.opts; opts.TCPKeepAlive > 0 {
			idle := opts.TCPKeepAlive
			intvl := opts.TCPKeepInterval
			if intvl == 0 {
				intvl = opts.TCPKeepAlive / 5
			}

			cnt := opts.TCPKeepCount
			if cnt == 0 {
				cnt = 5
			}

			if err = c.SetKeepAlive(true, idle, intvl, cnt); err != nil {
				return
			}
		}

		el.ch <- &openConn{
			c:  c,
			cb: func() { close(connOpened) },
		}

		xgoroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, readErr := nc.Read(buffer[:])
				if readErr != nil {
					el.ch <- &netErr{c, readErr}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		})

		gc = c

	case *net.UnixConn:
		c := newStreamConn(el, nc, ctx)

		el.ch <- &openConn{
			c:  c,
			cb: func() { close(connOpened) },
		}

		xgoroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, readErr := nc.Read(buffer[:])
				if readErr != nil {
					el.ch <- &netErr{c, readErr}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		})

		gc = c

	case *net.UDPConn:
		c := newUDPConn(el, nil, nc, nc.LocalAddr(), nc.RemoteAddr(), ctx)

		el.ch <- &openConn{
			c:  c,
			cb: func() { close(connOpened) },
		}

		xgoroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, readErr := nc.Read(buffer[:])
				if readErr != nil {
					el.ch <- &netErr{c, readErr}
					return
				}

				uc := newUDPConn(el, nil, nc, nc.LocalAddr(), nc.RemoteAddr(), ctx)
				el.ch <- packUDPConn(uc, buffer[:n])
			}
		})

		gc = c

	default:
		return nil, xerrors.ErrUnsupportedProtocol
	}

	<-connOpened
	return
}
