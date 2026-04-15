//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xnetpoll"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/utils/xsocket"
	"github.com/xbaseio/xbase/xerrors"
)

//
// =========================
// engine 定义
// =========================
//

// engine 是整个 xnet 的核心调度器。
type engine struct {
	listeners    map[int]*listener // 监听器集合（fd -> listener）
	opts         *Options          // 配置
	ingress      *eventloop        // 主 reactor（用于 accept）
	eventLoops   loadBalancer      // 子 reactor / worker loops
	inShutdown   atomic.Bool       // 是否已进入关闭状态
	turnOff      context.CancelFunc
	eventHandler EventHandler // 用户回调

	concurrency struct {
		*errgroup.Group
		ctx context.Context
	}
}

func (eng *engine) isShutdown() bool {
	return eng.inShutdown.Load()
}

//
// =========================
// 生命周期控制
// =========================
//

// shutdown 触发引擎关闭（只负责发信号）
func (eng *engine) shutdown(err error) {
	if err != nil && !xerrors.Is(err, xerrors.ErrEngineShutdown) {
		log.Errorf("engine is being shutdown with error: %v", err)
	}
	eng.turnOff()
}

// closeEventLoops 关闭所有 poller + listener
func (eng *engine) closeEventLoops() {
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		for _, ln := range el.listeners {
			ln.close()
		}
		_ = el.poller.Close()
		return true
	})

	if eng.ingress != nil {
		for _, ln := range eng.listeners {
			ln.close()
		}
		if err := eng.ingress.poller.Close(); err != nil {
			log.Errorf("failed to close poller when stopping engine: %v", err)
		}
	}
}

//
// =========================
// 单 reactor 模式（ReusePort）
// =========================
//

// runEventLoops：每个 loop 都持有独立 listener（SO_REUSEPORT）
func (eng *engine) runEventLoops(ctx context.Context, numEventLoop int) error {
	var tickerLoop *eventloop
	lns := eng.listeners

	for i := 0; i < numEventLoop; i++ {

		// 非第一个 loop，需要复制 listener（多 accept）
		if i > 0 {
			lns = make(map[int]*listener, len(eng.listeners))
			for _, l := range eng.listeners {
				ln, err := initListener(l.network, l.address, eng.opts)
				if err != nil {
					return err
				}
				lns[ln.fd] = ln
			}
		}

		poller, err := xnetpoll.OpenPoller()
		if err != nil {
			return err
		}

		el := &eventloop{
			listeners:    lns,
			engine:       eng,
			poller:       poller,
			buffer:       make([]byte, eng.opts.ReadBufferCap),
			eventHandler: eng.eventHandler,
		}
		el.connections.init()

		// 注册 accept 事件
		for _, ln := range lns {
			if err = el.poller.AddRead(ln.packPollAttachment(el.accept), false); err != nil {
				return err
			}
		}

		eng.eventLoops.register(el)

		// ticker 只在第一个 loop 启动
		if eng.opts.Ticker && el.idx == 0 {
			tickerLoop = el
		}
	}

	// 启动所有 event-loop
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		eng.concurrency.Go(el.run)
		return true
	})

	// 启动 ticker
	if tickerLoop != nil {
		eng.concurrency.Go(func() error {
			tickerLoop.ticker(ctx)
			return nil
		})
	}

	return nil
}

//
// =========================
// 多 reactor 模式（主从）
// =========================
//

// activateReactors：主 reactor accept，子 reactor 处理 IO
func (eng *engine) activateReactors(ctx context.Context, numEventLoop int) error {

	// 创建子 reactor（worker loops）
	for i := 0; i < numEventLoop; i++ {
		poller, err := xnetpoll.OpenPoller()
		if err != nil {
			return err
		}

		el := &eventloop{
			listeners:    eng.listeners,
			engine:       eng,
			poller:       poller,
			buffer:       make([]byte, eng.opts.ReadBufferCap),
			eventHandler: eng.eventHandler,
		}
		el.connections.init()

		eng.eventLoops.register(el)
	}

	// 启动 worker loops
	eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		eng.concurrency.Go(el.orbit)
		return true
	})

	// 主 reactor（专门 accept）
	poller, err := xnetpoll.OpenPoller()
	if err != nil {
		return err
	}

	mainLoop := &eventloop{
		listeners:    eng.listeners,
		idx:          -1,
		engine:       eng,
		poller:       poller,
		eventHandler: eng.eventHandler,
	}

	for _, ln := range eng.listeners {
		if err = mainLoop.poller.AddRead(
			ln.packPollAttachment(mainLoop.accept0),
			true,
		); err != nil {
			return err
		}
	}

	eng.ingress = mainLoop

	// 启动主 reactor
	eng.concurrency.Go(mainLoop.rotate)

	// ticker
	if eng.opts.Ticker {
		eng.concurrency.Go(func() error {
			eng.ingress.ticker(ctx)
			return nil
		})
	}

	return nil
}

//
// =========================
// 启动 / 停止
// =========================
//

// start 根据配置选择 reactor 模型
func (eng *engine) start(ctx context.Context, numEventLoop int) error {
	if eng.opts.ReusePort {
		return eng.runEventLoops(ctx, numEventLoop)
	}
	return eng.activateReactors(ctx, numEventLoop)
}

// stop 等待关闭信号并执行优雅关闭
func (eng *engine) stop(ctx context.Context, s Engine) {
	<-ctx.Done()

	// 用户回调
	eng.eventHandler.OnShutdown(s)

	// 通知所有 event-loop 退出
	eng.eventLoops.iterate(func(i int, el *eventloop) bool {
		err := el.poller.Trigger(
			xqueue.HighPriority,
			func(_ any) error { return xerrors.ErrEngineShutdown },
			nil,
		)
		if err != nil {
			log.Errorf("failed to enqueue shutdown signal for event-loop(%d): %v", i, err)
		}
		return true
	})

	// 通知主 reactor
	if eng.ingress != nil {
		if err := eng.ingress.poller.Trigger(
			xqueue.HighPriority,
			func(_ any) error { return xerrors.ErrEngineShutdown },
			nil,
		); err != nil {
			log.Errorf("failed to enqueue shutdown signal for main event-loop: %v", err)
		}
	}

	// 等待全部退出
	if err := eng.concurrency.Wait(); err != nil {
		log.Errorf("engine shutdown error: %v", err)
	}

	// 关闭资源
	eng.closeEventLoops()

	// 标记状态
	eng.inShutdown.Store(true)
}

//
// =========================
// 入口函数
// =========================
//

// run 启动整个 xnet 引擎
func run(eventHandler EventHandler, listeners []*listener, options *Options, addrs []string) error {
	numEventLoop := determineEventLoops(options)

	log.Infof(
		"Launching xnet with %d event-loops, listening on: %s",
		numEventLoop,
		strings.Join(addrs, " | "),
	)

	// listener map
	lns := make(map[int]*listener, len(listeners))
	for _, ln := range listeners {
		lns[ln.fd] = ln
	}

	// context + errgroup
	rootCtx, shutdown := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(rootCtx)

	eng := engine{
		listeners:    lns,
		opts:         options,
		turnOff:      shutdown,
		eventHandler: eventHandler,
		concurrency: struct {
			*errgroup.Group
			ctx context.Context
		}{eg, ctx},
	}

	// 负载均衡策略
	switch options.LB {
	case RoundRobin:
		eng.eventLoops = new(roundRobinLoadBalancer)
	case LeastConnections:
		eng.eventLoops = new(leastConnectionsLoadBalancer)
	case SourceAddrHash:
		eng.eventLoops = new(sourceAddrHashLoadBalancer)
	}

	e := Engine{&eng}

	// 启动回调
	switch eng.eventHandler.OnBoot(e) {
	case None, Close:
	case Shutdown:
		return nil
	}

	// 启动 engine
	if err := eng.start(ctx, numEventLoop); err != nil {
		eng.closeEventLoops()
		log.Errorf("xnet engine is stopping with error: %v", err)
		return err
	}

	defer eng.stop(rootCtx, e)

	// 注册全局 engine
	for _, addr := range addrs {
		allEngines.Store(addr, &eng)
	}

	return nil
}

//
// =========================
// 辅助函数
// =========================
//

// setKeepAlive 设置 TCP keepalive 参数
func setKeepAlive(fd int, enabled bool, idle, intvl time.Duration, cnt int) error {
	if intvl == 0 {
		intvl = idle / 5
	}
	if cnt == 0 {
		cnt = 5
	}
	return xsocket.SetKeepAlive(
		fd,
		enabled,
		int(idle.Seconds()),
		int(intvl.Seconds()),
		cnt,
	)
}
