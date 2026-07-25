package node

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/component"
	"github.com/xbaseio/xbase/core/info"
	"github.com/xbaseio/xbase/internal/transporter/node"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/transport"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xlog"
	"golang.org/x/sync/errgroup"
)

type HookHandler func(proxy *Proxy)

type serviceEntity struct {
	name     string // 服务名称;用于定位服务发现
	desc     any    // 服务描述(grpc为desc描述对象; rpcx为服务路径)
	provider any    // 服务提供者
}

type Node struct {
	component.Base
	opts        *options
	ctx         context.Context
	cancel      context.CancelFunc
	state       atomic.Int32
	evtPool     *sync.Pool
	reqPool     *sync.Pool
	router      *Router
	trigger     *Trigger
	proxy       *Proxy
	services    []*serviceEntity
	instances   []*registry.ServiceInstance
	linker      *node.Server
	fnChan      chan func()
	scheduler   *Scheduler
	transporter transport.Server
	wg          *sync.WaitGroup
	rw          sync.RWMutex
	hooks       map[cluster.Hook][]HookHandler
}

func NewNode(opts ...Option) *Node {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	n := &Node{}
	n.opts = o
	n.ctx, n.cancel = context.WithCancel(o.ctx)
	n.proxy = newProxy(n)
	n.router = newRouter(n)
	n.trigger = newTrigger(n)
	n.scheduler = newScheduler(n)
	n.hooks = make(map[cluster.Hook][]HookHandler)
	n.services = make([]*serviceEntity, 0)
	n.instances = make([]*registry.ServiceInstance, 0)
	n.fnChan = make(chan func(), 4096)
	n.state.Store(int32(cluster.Shut))
	n.wg = &sync.WaitGroup{}
	n.evtPool = &sync.Pool{New: func() any {
		evt := &event{}
		evt.node = n
		evt.actor.Store((*Actor)(nil))

		return evt
	}}
	n.reqPool = &sync.Pool{New: func() any {
		req := &request{}
		req.node = n
		req.message = &cluster.Message{}
		req.actor.Store((*Actor)(nil))

		return req
	}}

	return n
}

// Name 组件名称
func (n *Node) Name() string {
	return n.opts.name
}

// Init 初始化节点
func (n *Node) Init() {
	if n.opts.id == "" {
		xlog.Logger().Fatal("instance id can not be empty")
	}

	if n.opts.name == "" {
		xlog.Logger().Fatal("instance name can not be empty")
	}

	if n.opts.gameID <= cluster.GateGameID {
		xlog.Logger().Fatal("node game id must be greater than gate game id")
	}

	if n.opts.codec == nil {
		xlog.Logger().Fatal("codec component is not injected")
	}

	if n.opts.locator == nil {
		xlog.Logger().Fatal("locator component is not injected")
	}

	if n.opts.registry == nil {
		xlog.Logger().Fatal("registry component is not injected")
	}

	n.runHookFunc(cluster.Init)

	if n.router.StatefulRouteCount() > 0 {
		xlog.Sugar().Warnf("node %s has %d stateful routes; ensure BindNode is called before stateful traffic",
			n.opts.name, n.router.StatefulRouteCount())
	}
}

// Start 启动节点
func (n *Node) Start() {
	if !n.state.CompareAndSwap(int32(cluster.Shut), int32(cluster.Work)) {
		return
	}

	n.startLinkServer()

	n.startTransportServer()

	n.registerServiceInstances()

	n.proxy.watch()

	n.watchVersionRetire()

	go n.startDispatch()

	n.printInfo()

	n.runHookFunc(cluster.Start)
}

// Close 关闭节点
func (n *Node) Close() {
	if !n.state.CompareAndSwap(int32(cluster.Work), int32(cluster.Hang)) {
		if !n.state.CompareAndSwap(int32(cluster.Busy), int32(cluster.Hang)) {
			return
		}
	}

	n.refreshServiceInstances()

	n.runHookFunc(cluster.Close)

	n.stopLinkServer()

	n.stopTransportServer()

	n.wg.Wait()
}

// Destroy 销毁节点服务器
func (n *Node) Destroy() {
	if !n.state.CompareAndSwap(int32(cluster.Hang), int32(cluster.Shut)) {
		return
	}

	n.runHookFunc(cluster.Destroy)

	n.deregisterServiceInstances()

	n.stopLinkServer()

	n.stopTransportServer()

	n.router.close()

	n.trigger.close()

	close(n.fnChan)

	n.cancel()
}

// Proxy 获取节点代理
func (n *Node) Proxy() *Proxy {
	return n.proxy
}

// 启动 dispatch worker 池
func (n *Node) startDispatch() {
	go n.dispatchFn()

	for range n.opts.eventWorkers {
		go n.dispatchEvents()
	}

	for range n.opts.requestWorkers {
		go n.dispatchRequests()
	}
}

func (n *Node) dispatchFn() {
	for handle := range n.fnChan {
		xcall.Call(handle)
		n.doneWait()
	}
}

func (n *Node) dispatchEvents() {
	for evt := range n.trigger.receive() {
		n.trigger.handle(evt)
	}
}

func (n *Node) dispatchRequests() {
	for req := range n.router.receive() {
		n.router.handle(req)
	}
}

// 启动连接服务器
func (n *Node) startLinkServer() {
	linker, err := node.NewServer(&provider{node: n}, &node.ServerOptions{
		Addr:   n.opts.addr,
		Expose: n.opts.expose,
	})
	if err != nil {
		xlog.Sugar().Fatalf("link server create failed: %v", err)
	}

	n.linker = linker

	go func() {
		if err = n.linker.Start(); err != nil {
			xlog.Sugar().Fatalf("link server start failed: %v", err)
		}
	}()

	if err = cluster.WaitForTCPListen(n.linker.ListenAddr(), defaultTimeout); err != nil {
		xlog.Sugar().Fatalf("link server listen timeout: %v", err)
	}
}

// 停止连接服务器
func (n *Node) stopLinkServer() {
	if err := n.linker.Stop(); err != nil {
		xlog.Sugar().Errorf("link server stop failed: %v", err)
	}
}

// 启动传输服务器
func (n *Node) startTransportServer() {
	if n.opts.transporter == nil {
		return
	}

	n.opts.transporter.SetDefaultDiscovery(n.opts.registry)

	if len(n.services) == 0 {
		return
	}

	transporter, err := n.opts.transporter.NewServer()
	if err != nil {
		xlog.Sugar().Fatalf("transport server create failed: %v", err)
	}

	n.transporter = transporter

	for _, entity := range n.services {
		if err = n.transporter.RegisterService(entity.desc, entity.provider); err != nil {
			xlog.Sugar().Fatalf("register service failed: %v", err)
		}
	}

	go func() {
		if err = n.transporter.Start(); err != nil {
			xlog.Sugar().Fatalf("transport server start failed: %v", err)
		}
	}()

	if err = cluster.WaitForTCPListen(n.transporter.Addr(), defaultTimeout); err != nil {
		xlog.Sugar().Fatalf("transport server listen timeout: %v", err)
	}
}

// 停止传输服务器
func (n *Node) stopTransportServer() {
	if n.transporter == nil {
		return
	}

	if err := n.transporter.Stop(); err != nil {
		xlog.Sugar().Errorf("transport server stop failed: %v", err)
	}
}

// 注册服务实例
func (n *Node) registerServiceInstances() {
	routes := n.router.CollectRoutes()
	events := make([]int, 0, len(n.trigger.events))

	for evt := range n.trigger.events {
		events = append(events, int(evt))
	}

	n.instances = append(n.instances, &registry.ServiceInstance{
		ID:       n.opts.id,
		Name:     cluster.Node.String(),
		Kind:     cluster.Node.String(),
		Alias:    n.opts.name,
		State:    n.getState().String(),
		Events:   events,
		Routes:   routes,
		Endpoint: n.linker.Endpoint().String(),
		Weight:   n.opts.weight,
		Metadata: mergeServiceMetadata(n.opts.metadata, n.opts.serviceStatus),
		GameID:   n.opts.gameID,
		Version:  n.opts.version,
	})

	if n.transporter != nil {
		services := make([]string, 0, len(n.services))
		for _, item := range n.services {
			services = append(services, item.name)
		}

		n.instances = append(n.instances, &registry.ServiceInstance{
			ID:       n.opts.id,
			Name:     cluster.Mesh.String(),
			Kind:     cluster.Mesh.String(),
			Alias:    n.opts.name,
			State:    n.getState().String(),
			Services: services,
			Endpoint: n.transporter.Endpoint().String(),
			Weight:   n.opts.weight,
			Metadata: mergeServiceMetadata(n.opts.metadata, n.opts.serviceStatus),
			Version:  n.opts.version,
		})
	}

	if err := n.doRegisterServiceInstances(); err != nil {
		xlog.Sugar().Fatalf("register cluster instances failed: %v", err)
	}
}

// 刷新服务实例状态
func (n *Node) refreshServiceInstances() {
	if err := n.doRefreshServiceInstances(); err != nil {
		xlog.Sugar().Errorf("refresh cluster instances failed: %v", err)
	}
}

// 解注册服务实例
func (n *Node) deregisterServiceInstances() {
	eg, ctx := errgroup.WithContext(n.ctx)
	for i := range n.instances {
		instance := n.instances[i]
		eg.Go(func() error {
			tctx, tcancel := context.WithTimeout(ctx, defaultTimeout)
			defer tcancel()
			return n.opts.registry.Deregister(tctx, instance)
		})
	}

	if err := eg.Wait(); err != nil {
		xlog.Sugar().Errorf("deregister cluster instances failed: %v", err)
	}
}

// 执行注册操作
func (n *Node) doRegisterServiceInstances() error {
	eg, ctx := errgroup.WithContext(n.ctx)

	for i := range n.instances {
		instance := n.instances[i]
		eg.Go(func() error {
			tctx, tcancel := context.WithTimeout(ctx, defaultTimeout)
			defer tcancel()
			return n.opts.registry.Register(tctx, instance)
		})
	}

	return eg.Wait()
}

// 执行刷新实例状态操作
func (n *Node) doRefreshServiceInstances() error {
	for _, instance := range n.instances {
		instance.State = n.getState().String()
	}

	return n.doRegisterServiceInstances()
}

// 获取状态
func (n *Node) getState() cluster.State {
	return cluster.State(n.state.Load())
}

// 更新状态
func (n *Node) setState(state cluster.State) error {
	n.state.Store(int32(state))

	return n.doRefreshServiceInstances()
}

// 执行钩子函数
func (n *Node) runHookFunc(hook cluster.Hook) {
	n.rw.RLock()

	if handlers, ok := n.hooks[hook]; ok {
		wg := &sync.WaitGroup{}
		wg.Add(len(handlers))

		for i := range handlers {
			handler := handlers[i]
			xcall.Go(func() {
				handler(n.proxy)
				wg.Done()
			})
		}

		n.rw.RUnlock()

		wg.Wait()
	} else {
		n.rw.RUnlock()
	}
}

// 添加钩子监听器
func (n *Node) addHookListener(hook cluster.Hook, handler HookHandler) {
	switch hook {
	case cluster.Destroy:
		n.rw.Lock()
		n.hooks[hook] = append(n.hooks[hook], handler)
		n.rw.Unlock()
	default:
		if n.getState() == cluster.Shut {
			n.hooks[hook] = append(n.hooks[hook], handler)
		} else {
			xlog.Sugar().Warnf("server is working, can't add hook handler")
		}
	}
}

// 添加服务提供者
func (n *Node) addServiceProvider(name string, desc, provider any) {
	if n.getState() == cluster.Shut {
		n.services = append(n.services, &serviceEntity{
			name:     name,
			desc:     desc,
			provider: provider,
		})
	} else {
		xlog.Sugar().Warnf("server is working, can't add service provider")
	}
}

// 打印组件信息
func (n *Node) printInfo() {
	infos := make([]string, 0, 8)
	infos = append(infos, fmt.Sprintf("ID: %s", n.opts.id))
	infos = append(infos, fmt.Sprintf("Name: %s", n.Name()))
	infos = append(infos, fmt.Sprintf("Link: %s", n.linker.ExposeAddr()))
	infos = append(infos, fmt.Sprintf("Codec: %s", n.opts.codec.Name()))
	infos = append(infos, fmt.Sprintf("Locator: %s", n.opts.locator.Name()))
	infos = append(infos, fmt.Sprintf("Registry: %s", n.opts.registry.Name()))
	infos = append(infos, fmt.Sprintf("NodeKind: %s", n.opts.nodeKind.String()))
	if n.opts.encryptor != nil {
		infos = append(infos, fmt.Sprintf("Encryptor: %s", n.opts.encryptor.Name()))
	} else {
		infos = append(infos, "Encryptor: -")
	}

	if n.opts.transporter != nil {
		infos = append(infos, fmt.Sprintf("Transporter: %s", n.opts.transporter.Name()))
	} else {
		infos = append(infos, "Transporter: -")
	}

	info.PrintBoxInfo("Node", infos...)
}

func (n *Node) doneWait() {
	if n.getState() != cluster.Shut {
		n.wg.Done()
	}
}

func (n *Node) addWait() {
	if n.getState() != cluster.Shut {
		n.wg.Add(1)
	}
}

func mergeServiceMetadata(metadata map[string]string, status registry.ServiceStatus) map[string]string {
	merged := make(map[string]string, len(metadata)+1)
	maps.Copy(merged, metadata)
	merged[registry.MetadataServiceStatusKey] = string(status)

	return merged
}
