package node

import (
	"context"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xerrors"
)

type RouteHandler func(ctx Context)

// RouteOptions 路由选项
type RouteOptions struct {
	// 是否内部的路由，默认非内部
	// 外部路由可在客户端、网关、节点间进行消息流转
	// 内部路由仅限于在节点间进行消息流转
	Internal bool

	// 是否有状态路由，默认无状态
	// 无状态路由消息会根据负载均衡策略分配到不同的节点服务器进行处理
	// 有状态路由消息会在绑定节点服务器后，固定路由到绑定的节点服务器进行处理
	Stateful bool

	// 是否授权路由，默认非授权
	// 授权路由在集群间流转时必需附带UID信息，否则无法进行路由投递
	// 该参数可在网关层对未授权连接进行提前拦截，降低节点服对于攻击处理的压力
	Authorized bool

	// 路由中间件
	Middlewares []MiddlewareHandler
}

var (
	InternalRoute   = RouteOptions{Internal: true}
	StatefulRoute   = RouteOptions{Stateful: true}
	AuthorizedRoute = RouteOptions{Authorized: true}
)

type Router struct {
	node                *Node
	routes              map[int32]*routeEntity
	reqChan             chan *request
	preRouteHandler     RouteHandler
	postRouteHandler    RouteHandler
	defaultRouteHandler RouteHandler
}

type routeEntity struct {
	messageID int32        // 消息ID
	handler   RouteHandler // 路由处理器
	options   RouteOptions // 路由选项
}

func newRouter(node *Node) *Router {
	return &Router{
		node:    node,
		routes:  make(map[int32]*routeEntity),
		reqChan: make(chan *request, 10240),
	}
}

// AddRouteHandler 添加路由处理器，按 MessageID 注册（GameID 已在网关层完成节点选址）
func (r *Router) AddRouteHandler(messageID int32, handler RouteHandler, opts ...RouteOptions) {
	if r.node.getState() != cluster.Shut {
		log.Warnf("the node server is working, can't add route handler")
		return
	}

	entity := &routeEntity{
		messageID: messageID,
		handler:   handler,
	}
	if len(opts) > 0 {
		entity.options = opts[0]
	}

	r.routes[messageID] = entity
}

// SetDefaultRouteHandler 设置默认路由处理器，所有未注册的路由均走默认路由处理器
func (r *Router) SetDefaultRouteHandler(handler RouteHandler) {
	if r.node.getState() != cluster.Shut {
		log.Warnf("the node server is working, can't set default route handler")
		return
	}

	r.defaultRouteHandler = handler
}

// HasDefaultRouteHandler 是否存在默认路由处理器
func (r *Router) HasDefaultRouteHandler() bool {
	return r.defaultRouteHandler != nil
}

// SetPreRouteHandler 设置前置路由处理器
func (r *Router) SetPreRouteHandler(handler RouteHandler) {
	if r.node.getState() != cluster.Shut {
		log.Warnf("the node server is working, can't set pre-route handler")
		return
	}

	r.preRouteHandler = handler
}

// SetPostRouteHandler 设置后置路由处理器
func (r *Router) SetPostRouteHandler(handler RouteHandler) {
	if r.node.getState() != cluster.Shut {
		log.Warnf("the node server is working, can't set post-route handler")
		return
	}

	r.postRouteHandler = handler
}

// CheckRouteStateful 是否为有状态路由
func (r *Router) CheckRouteStateful(messageID int32) (stateful bool, exist bool) {
	if entity, ok := r.routes[messageID]; ok {
		exist, stateful = ok, entity.options.Stateful
	}
	return
}

// CheckRouteAuthorized 是否为需授权路由
func (r *Router) CheckRouteAuthorized(messageID int32) (authorized bool, exist bool) {
	if entity, ok := r.routes[messageID]; ok {
		exist, authorized = ok, entity.options.Authorized
	}
	return
}

// CollectRoutes 收集路由元数据用于注册中心同步
func (r *Router) CollectRoutes() []registry.Route {
	routes := make([]registry.Route, 0, len(r.routes))
	for _, entity := range r.routes {
		routes = append(routes, registry.Route{
			ID:         entity.messageID,
			Internal:   entity.options.Internal,
			Stateful:   entity.options.Stateful,
			Authorized: entity.options.Authorized,
		})
	}
	return routes
}

// StatefulRouteCount 有状态路由数量
func (r *Router) StatefulRouteCount() int {
	n := 0
	for _, entity := range r.routes {
		if entity.options.Stateful {
			n++
		}
	}
	return n
}

// Group 路由组
func (r *Router) Group(groups ...func(group *RouterGroup)) *RouterGroup {
	group := &RouterGroup{
		router:      r,
		middlewares: make([]MiddlewareHandler, 0),
	}

	for _, fn := range groups {
		fn(group)
	}

	return group
}

func (r *Router) deliver(gid, nid, pid string, cid, uid int64, seq, gameID, messageID int32, data any) error {
	req := r.node.reqPool.Get().(*request)
	req.ctx = context.Background()
	req.gid = gid
	req.nid = nid
	req.pid = pid
	req.cid = cid
	req.uid = uid
	req.message.Seq = seq
	req.message.GameID = gameID
	req.message.MessageID = messageID
	req.message.Data = data

	timeout := r.node.opts.deliverTimeout
	if timeout <= 0 {
		r.reqChan <- req
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case r.reqChan <- req:
		return nil
	case <-timer.C:
		req.reset()
		r.node.reqPool.Put(req)
		log.Warnf("node request queue full, drop message: %d uid: %d", messageID, uid)
		return xerrors.ErrDeliverQueueFull
	}
}

func (r *Router) receive() <-chan *request {
	return r.reqChan
}

func (r *Router) close() {
	close(r.reqChan)
}

func (r *Router) handle(req *request) {
	version := req.incrVersion()

	route, ok := r.routes[req.message.MessageID]
	if !ok && r.defaultRouteHandler == nil {
		req.compareVersionRecycle(version)
		log.Warnf("message routing does not register handler function, message: %v", req.message.MessageID)
		return
	}

	if r.preRouteHandler != nil {
		xcall.Call(func() { r.preRouteHandler(req) })
	}

	if ok {
		if len(route.options.Middlewares) > 0 {
			middleware := &Middleware{index: -1, middlewares: route.options.Middlewares, routeHandler: route.handler}
			middleware.Next(req)
			return
		} else {
			xcall.Call(func() { route.handler(req) })
		}
	} else {
		xcall.Call(func() { r.defaultRouteHandler(req) })
	}

	req.compareVersionExecDefer(version)

	req.compareVersionRecycle(version)
}

type RouterGroup struct {
	router      *Router
	middlewares []MiddlewareHandler
}

// Middleware 添加中间件
func (g *RouterGroup) Middleware(middlewares ...MiddlewareHandler) *RouterGroup {
	g.middlewares = append(g.middlewares, middlewares...)

	return g
}

// AddRouteHandler 添加路由处理器
func (g *RouterGroup) AddRouteHandler(messageID int32, handler RouteHandler, opts ...RouteOptions) *RouterGroup {
	var options RouteOptions

	if len(opts) > 0 {
		options = opts[0]
		options.Middlewares = make([]MiddlewareHandler, len(g.middlewares)+len(opts[0].Middlewares))
		copy(options.Middlewares, g.middlewares)
		copy(options.Middlewares[len(g.middlewares):], opts[0].Middlewares)
	} else {
		options = RouteOptions{}
		options.Middlewares = make([]MiddlewareHandler, len(g.middlewares))
		copy(options.Middlewares, g.middlewares)
	}

	g.router.AddRouteHandler(messageID, handler, options)

	return g
}
