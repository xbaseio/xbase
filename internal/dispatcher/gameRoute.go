package dispatcher

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/endpoint"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/xerrors"
)

type GameRoute struct {
	abstract
	group      string
	counter    atomic.Uint64
	dispatcher *Dispatcher
	gameID     int32
}

func newRoute_001(dispatcher *Dispatcher, group string, gameID int32) *GameRoute {
	return &GameRoute{
		gameID:     gameID,
		group:      group,
		dispatcher: dispatcher,
		abstract:   newAbstract(),
	}
}

func (r *GameRoute) ID() int32     { return r.gameID }
func (r *GameRoute) Group() string { return r.group }

func (r *GameRoute) FindEndpoint(insID ...string) (*endpoint.Endpoint, error) {
	return r.FindEndpointForServiceStatus(registry.ServiceStatusNormal, insID...)
}

func (r *GameRoute) FindEndpointForUser(allowTest bool, insID ...string) (*endpoint.Endpoint, error) {
	if allowTest {
		return r.FindEndpointForServiceStatus(registry.ServiceStatusTest, insID...)
	}

	return r.FindEndpointForServiceStatus(registry.ServiceStatusNormal, insID...)
}

func (r *GameRoute) FindEndpointForServiceStatus(status registry.ServiceStatus, insID ...string) (*endpoint.Endpoint, error) {
	se, err := r.findServiceEndpointForServiceStatus(status, insID...)
	if err != nil {
		return nil, err
	}

	return se.endpoint, nil
}

func (r *GameRoute) findServiceEndpointForServiceStatus(status registry.ServiceStatus, insID ...string) (*serviceEndpoint, error) {
	if len(insID) == 0 || insID[0] == "" {
		switch r.dispatcher.dispatch {
		case cluster.RoundRobin:
			return r.roundRobinDispatch(status)
		case cluster.WeightRoundRobin:
			return r.weightRoundRobinDispatch(status)
		default:
			return r.randomDispatch(status)
		}
	}

	return r.directDispatch(insID[0])
}

func (r *GameRoute) directDispatch(insID string) (*serviceEndpoint, error) {
	sep, ok := r.endpoints5[insID]
	if !ok {
		return nil, xerrors.ErrNotFoundEndpoint
	}
	return sep, nil
}

func (r *GameRoute) randomDispatch(status registry.ServiceStatus) (*serviceEndpoint, error) {
	if endpoints := r.balanceEndpoints(status); len(endpoints) > 0 {
		return endpoints[rand.IntN(len(endpoints))], nil
	}
	return nil, xerrors.ErrNotFoundEndpoint
}

func (r *GameRoute) roundRobinDispatch(status registry.ServiceStatus) (*serviceEndpoint, error) {
	if endpoints := r.balanceEndpoints(status); len(endpoints) > 0 {
		index := int(r.counter.Add(1) % uint64(len(endpoints)))
		return endpoints[index], nil
	}
	return nil, xerrors.ErrNotFoundEndpoint
}

func (r *GameRoute) weightRoundRobinDispatch(status registry.ServiceStatus) (*serviceEndpoint, error) {
	var (
		selected    *serviceEndpoint
		totalWeight int
	)

	if endpoints := r.balanceEndpoints(status); len(endpoints) > 0 {
		for i := range endpoints {
			se := endpoints[i]
			se.currWeight += se.weight
			totalWeight += se.weight
			if selected == nil || se.currWeight > selected.currWeight {
				selected = se
			}
		}

		if selected != nil {
			selected.currWeight -= totalWeight
			return selected, nil
		}
	}

	return nil, xerrors.ErrNotFoundEndpoint
}

func (r *GameRoute) balanceEndpoints(status registry.ServiceStatus) []*serviceEndpoint {
	for _, preferred := range registry.PreferredServiceStatuses(status) {
		switch preferred {
		case registry.ServiceStatusTest:
			if len(r.endpoints6) > 0 {
				return r.endpoints6
			}
			if len(r.endpoints7) > 0 {
				return r.endpoints7
			}
		case registry.ServiceStatusGray:
			if len(r.endpoints8) > 0 {
				return r.endpoints8
			}
			if len(r.endpoints9) > 0 {
				return r.endpoints9
			}
		default:
			if len(r.endpoints1) > 0 {
				return r.endpoints1
			}
			if len(r.endpoints2) > 0 {
				return r.endpoints2
			}
		}
	}

	return nil
}
