package gate

import (
	"context"
	"sync"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/registry"
)

type routePolicy struct {
	authorized bool
	stateful   bool
}

type routeWatcher struct {
	ctx      context.Context
	registry registry.Registry
	rw       sync.RWMutex
	routes   map[int32]map[int32]routePolicy
}

func newRouteWatcher(ctx context.Context, reg registry.Registry) *routeWatcher {
	return &routeWatcher{
		ctx:      ctx,
		registry: reg,
		routes:   make(map[int32]map[int32]routePolicy),
	}
}

func (w *routeWatcher) start() {
	if w.registry == nil {
		return
	}

	go w.watch()
}

func (w *routeWatcher) watch() {
	tctx, cancel := context.WithTimeout(w.ctx, defaultTimeout)
	instances, err := w.registry.Services(tctx, cluster.Node.String())
	cancel()
	if err == nil {
		w.replace(instances)
	}

	tctx, cancel = context.WithTimeout(w.ctx, defaultTimeout)
	watcher, err := w.registry.Watch(tctx, cluster.Node.String())
	cancel()
	if err != nil {
		log.Warnf("gate route watch failed: %v", err)
		return
	}

	defer watcher.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		instances, err := watcher.Next()
		if err != nil {
			continue
		}

		w.replace(instances)
	}
}

func (w *routeWatcher) replace(instances []*registry.ServiceInstance) {
	routes := make(map[int32]map[int32]routePolicy)

	for _, ins := range instances {
		if ins == nil || ins.Kind != cluster.Node.String() {
			continue
		}

		gameRoutes, ok := routes[ins.GameID]
		if !ok {
			gameRoutes = make(map[int32]routePolicy)
			routes[ins.GameID] = gameRoutes
		}

		for _, route := range ins.Routes {
			cur := gameRoutes[route.ID]
			gameRoutes[route.ID] = routePolicy{
				authorized: cur.authorized || route.Authorized,
				stateful:   cur.stateful || route.Stateful,
			}
		}
	}

	w.rw.Lock()
	w.routes = routes
	w.rw.Unlock()
}

func (w *routeWatcher) requiresAuth(gameID, messageID int32) bool {
	w.rw.RLock()
	defer w.rw.RUnlock()

	gameRoutes, ok := w.routes[gameID]
	if !ok {
		return false
	}

	return gameRoutes[messageID].authorized
}
