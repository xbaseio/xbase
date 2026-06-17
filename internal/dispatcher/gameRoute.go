package dispatcher

import (
	"math/rand/v2"
	"sync/atomic"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/endpoint"
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
	return r.FindEndpointForUser(false, insID...)
}

func (r *GameRoute) FindEndpointForUser(allowTest bool, insID ...string) (*endpoint.Endpoint, error) {
	if len(insID) == 0 || insID[0] == "" {
		switch r.dispatcher.dispatch {
		case cluster.RoundRobin:
			return r.roundRobinDispatch(allowTest)
		case cluster.WeightRoundRobin:
			return r.weightRoundRobinDispatch(allowTest)
		default:
			return r.randomDispatch(allowTest)
		}
	}

	return r.directDispatch(insID[0])
}

func (r *GameRoute) directDispatch(insID string) (*endpoint.Endpoint, error) {
	sep, ok := r.endpoints5[insID]
	if !ok {
		return nil, xerrors.ErrNotFoundEndpoint
	}
	return sep.endpoint, nil
}

func (r *GameRoute) randomDispatch(allowTest bool) (*endpoint.Endpoint, error) {
	if endpoints := r.balanceEndpoints(allowTest); len(endpoints) > 0 {
		return endpoints[rand.IntN(len(endpoints))].endpoint, nil
	}
	return nil, xerrors.ErrNotFoundEndpoint
}

func (r *GameRoute) roundRobinDispatch(allowTest bool) (*endpoint.Endpoint, error) {
	if endpoints := r.balanceEndpoints(allowTest); len(endpoints) > 0 {
		index := int(r.counter.Add(1) % uint64(len(endpoints)))
		return endpoints[index].endpoint, nil
	}
	return nil, xerrors.ErrNotFoundEndpoint
}

func (r *GameRoute) weightRoundRobinDispatch(allowTest bool) (*endpoint.Endpoint, error) {
	var (
		selected    *serviceEndpoint
		totalWeight int
	)

	if endpoints := r.balanceEndpoints(allowTest); len(endpoints) > 0 {
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
			return selected.endpoint, nil
		}
	}

	return nil, xerrors.ErrNotFoundEndpoint
}

func (r *GameRoute) balanceEndpoints(allowTest bool) []*serviceEndpoint {
	if allowTest {
		if len(r.endpoints6) > 0 {
			return r.endpoints6
		}
		if len(r.endpoints7) > 0 {
			return r.endpoints7
		}
	}

	if len(r.endpoints1) > 0 {
		return r.endpoints1
	}
	if len(r.endpoints2) > 0 {
		return r.endpoints2
	}

	return nil
}
