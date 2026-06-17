package dispatcher

import (
	"sync"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/endpoint"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/xerrors"
)

type Dispatcher struct {
	dispatch  cluster.Dispatch
	rw        sync.RWMutex
	routes    map[int32]*GameRoute
	events    map[int]*Event
	endpoints map[string]*endpoint.Endpoint
	instances map[string]*registry.ServiceInstance
}

func NewDispatcher(dispatch cluster.Dispatch) *Dispatcher {
	return &Dispatcher{
		dispatch:  dispatch,
		routes:    make(map[int32]*GameRoute),
		events:    make(map[int]*Event),
		endpoints: make(map[string]*endpoint.Endpoint),
		instances: make(map[string]*registry.ServiceInstance),
	}
}

// FindEndpoint 查找服务端口
func (d *Dispatcher) FindEndpoint(insID string) (*endpoint.Endpoint, error) {
	d.rw.RLock()
	defer d.rw.RUnlock()

	ep, ok := d.endpoints[insID]
	if !ok {
		return nil, xerrors.ErrNotFoundEndpoint
	}

	return ep, nil
}

// Endpoints 获取所有端口
func (d *Dispatcher) Endpoints() map[string]*endpoint.Endpoint {
	d.rw.RLock()
	defer d.rw.RUnlock()

	return d.endpoints
}

// VisitEndpoints 迭代服务端口
func (d *Dispatcher) VisitEndpoints(fn func(insID string, ep *endpoint.Endpoint) bool) {
	d.rw.RLock()
	defer d.rw.RUnlock()

	for insID, ep := range d.endpoints {
		if !fn(insID, ep) {
			break
		}
	}
}

// FindGameRoute 按游戏ID查找节点路由
func (d *Dispatcher) FindGameRoute(gameID int32) (*GameRoute, error) {
	d.rw.RLock()
	defer d.rw.RUnlock()

	r, ok := d.routes[gameID]
	if !ok {
		return nil, xerrors.ErrNotFoundRoute
	}

	return r, nil
}

// FindRoute 查找节点路由
func (d *Dispatcher) FindRoute(gameID int32) (*GameRoute, error) {
	return d.FindGameRoute(gameID)
}

// FindEvent 查找节点事件
func (d *Dispatcher) FindEvent(event int) (*Event, error) {
	d.rw.RLock()
	defer d.rw.RUnlock()

	e, ok := d.events[event]
	if !ok {
		return nil, xerrors.ErrNotFoundEvent
	}

	return e, nil
}

// ReplaceServices 替换服务
func (d *Dispatcher) ReplaceServices(services ...*registry.ServiceInstance) {
	routes := make(map[int32]*GameRoute, len(services))
	events := make(map[int]*Event, len(services))
	endpoints := make(map[string]*endpoint.Endpoint)
	instances := make(map[string]*registry.ServiceInstance, len(services))
	maxVersionByGame := registry.MaxVersionForGame(services)
	maxVersionByGameAndStatus := registry.MaxVersionForGameByServiceStatus(services)
	maxVersionByGateAlias := registry.MaxVersionByKindAlias(services, cluster.Gate.String())

	for _, service := range services {
		ep, err := endpoint.ParseEndpoint(service.Endpoint)
		if err != nil {
			log.Errorf("service endpoint parse failed, insID: %s kind: %s name: %s alias: %s endpoint: %s err: %v",
				service.ID, service.Kind, service.Name, service.Alias, service.Endpoint, err)
			continue
		}

		endpoints[service.ID] = ep
		instances[service.ID] = service

		balance := true
		switch service.Kind {
		case cluster.Node.String():
			status := registry.ServiceStatusOf(service)
			if group, ok := maxVersionByGameAndStatus[service.GameID]; ok {
				balance = registry.IsLatestVersion(service, group[status])
			} else {
				balance = registry.IsLatestVersion(service, maxVersionByGame[service.GameID])
			}
		case cluster.Gate.String():
			balance = registry.IsLatestVersion(service, maxVersionByGateAlias[service.Alias])
		}

		se := &serviceEndpoint{
			insID:    service.ID,
			state:    service.State,
			status:   registry.ServiceStatusOf(service),
			endpoint: ep,
			weight:   service.Weight,
		}

		route, ok := routes[service.GameID]
		if !ok {
			route = newRoute_001(d, service.Alias, service.GameID)
			routes[service.GameID] = route
		}

		route.addServiceEndpoint(se, balance)

		for _, evt := range service.Events {
			event, ok := events[evt]
			if !ok {
				event = newEvent(evt)
				events[evt] = event
			}
			event.addServiceEndpoint(se, balance)
		}
	}

	d.rw.Lock()
	d.routes = routes
	d.events = events
	d.endpoints = endpoints
	d.instances = instances
	d.rw.Unlock()
}
