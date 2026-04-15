package xnet

import (
	"context"
	"strings"
	"sync/atomic"

	"golang.org/x/sync/errgroup"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/xerrors"
)

// engine 是 xnet 的核心调度器。
type engine struct {
	listeners     []*listener  // 监听器集合
	opts          *Options     // 引擎配置
	eventLoops    loadBalancer // 事件循环负载均衡器
	inShutdown    atomic.Bool  // 是否已经关闭
	beingShutdown atomic.Bool  // 是否正在关闭
	turnOff       context.CancelFunc
	eventHandler  EventHandler // 用户事件处理器

	concurrency struct {
		*errgroup.Group
		ctx context.Context
	}
}

// isShutdown 返回引擎是否已关闭。
func (eng *engine) isShutdown() bool {
	return eng.inShutdown.Load()
}

// shutdown 发出关闭信号。
func (eng *engine) shutdown(err error) {
	if err != nil && !xerrors.Is(err, xerrors.ErrEngineShutdown) {
		log.Errorf("engine is being shutdown with error: %v", err)
	}

	eng.turnOff()
	eng.beingShutdown.Store(true)
}

// closeEventLoops 通知所有事件循环退出，并关闭监听器。
func (eng *engine) closeEventLoops() {
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		el.ch <- xerrors.ErrEngineShutdown
		return true
	})

	for _, ln := range eng.listeners {
		ln.close()
	}
}

// start 启动事件循环、ticker 和所有监听协程。
func (eng *engine) start(ctx context.Context, numEventLoop int) error {
	var tickerLoop *eventloop

	// 启动 event-loop
	for i := 0; i < numEventLoop; i++ {
		el := eventloop{
			ch:           make(chan any, 1024),
			eng:          eng,
			connections:  make(map[*conn]struct{}),
			eventHandler: eng.eventHandler,
		}

		eng.eventLoops.register(&el)
		eng.concurrency.Go(el.run)

		if i == 0 && eng.opts.Ticker {
			tickerLoop = &el
		}
	}

	// 启动 ticker
	if tickerLoop != nil {
		eng.concurrency.Go(func() error {
			tickerLoop.ticker(ctx)
			return nil
		})
	}

	// 启动监听协程
	for _, ln := range eng.listeners {
		l := ln

		if l.pc != nil {
			eng.concurrency.Go(func() error {
				return eng.ListenUDP(l.pc)
			})
		} else {
			eng.concurrency.Go(func() error {
				return eng.listenStream(l.ln)
			})
		}
	}

	return nil
}

// stop 等待关闭信号并执行优雅停止。
func (eng *engine) stop(ctx context.Context, engine Engine) {
	<-ctx.Done()

	eng.eventHandler.OnShutdown(engine)
	eng.closeEventLoops()

	if err := eng.concurrency.Wait(); err != nil && !xerrors.Is(err, xerrors.ErrEngineShutdown) {
		log.Errorf("engine shutdown error: %v", err)
	}

	eng.inShutdown.Store(true)
}

// run 启动整个 xnet 引擎。
func run(eventHandler EventHandler, listeners []*listener, options *Options, addrs []string) error {
	numEventLoop := determineEventLoops(options)

	log.Infof(
		"Launching xnet with %d event-loops, listening on: %s",
		numEventLoop,
		strings.Join(addrs, " | "),
	)

	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)

	eng := engine{
		opts:         options,
		listeners:    listeners,
		turnOff:      shutdown,
		eventHandler: eventHandler,
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}

	// 选择负载均衡策略
	switch options.LB {
	case RoundRobin:
		eng.eventLoops = new(roundRobinLoadBalancer)

		// 当存在多个 listener 时，roundRobinLoadBalancer 不是并发安全的，
		// 因此回退到 leastConnectionsLoadBalancer。
		if len(listeners) > 1 {
			eng.eventLoops = new(leastConnectionsLoadBalancer)
		}

	case LeastConnections:
		eng.eventLoops = new(leastConnectionsLoadBalancer)

	case SourceAddrHash:
		eng.eventLoops = new(sourceAddrHashLoadBalancer)
	}

	engine := Engine{eng: &eng}

	switch eventHandler.OnBoot(engine) {
	case None, Close:
	case Shutdown:
		return nil
	}

	if err := eng.start(ctx, numEventLoop); err != nil {
		log.Errorf("xnet engine is stopping with error: %v", err)
		return err
	}
	defer eng.stop(rootCtx, engine)

	for _, addr := range addrs {
		allEngines.Store(addr, &eng)
	}

	return nil
}

/*
func (eng *engine) sendCmd(_ *asyncCmd, _ bool) error {
	return xerrors.ErrUnsupportedOp
}
*/
