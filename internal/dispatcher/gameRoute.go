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
	//route      registry.Route // 路由信息
	group      string        // 路由所属组
	counter    atomic.Uint64 // 轮询计数器
	dispatcher *Dispatcher   // 分发器
	gameID     int32         // 游戏ID
}

func newRoute_001(dispatcher *Dispatcher, group string, gameID int32) *GameRoute {
	return &GameRoute{
		gameID:     gameID,
		group:      group,
		dispatcher: dispatcher,
		abstract:   newAbstract(),
	}
}

// ID 获取路由ID
func (r *GameRoute) ID() int32 {
	return r.gameID
}

// Group 路由所属组
func (r *GameRoute) Group() string {
	return r.group
}

// FindEndpoint 查询路由服务端点
func (r *GameRoute) FindEndpoint(insID ...string) (*endpoint.Endpoint, error) {
	if len(insID) == 0 || insID[0] == "" {
		switch r.dispatcher.dispatch {
		case cluster.RoundRobin:
			return r.roundRobinDispatch()
		case cluster.WeightRoundRobin:
			return r.weightRoundRobinDispatch()
		default:
			return r.randomDispatch()
		}
	}

	return r.directDispatch(insID[0])
}

// 直接分配
func (r *GameRoute) directDispatch(insID string) (*endpoint.Endpoint, error) {
	sep, ok := r.endpoints5[insID]
	if !ok {
		return nil, xerrors.ErrNotFoundEndpoint
	}

	return sep.endpoint, nil
}

// 随机分配
func (r *GameRoute) randomDispatch() (*endpoint.Endpoint, error) {
	if n := len(r.endpoints1); n > 0 {
		return r.endpoints1[rand.IntN(n)].endpoint, nil
	}

	if n := len(r.endpoints2); n > 0 {
		return r.endpoints2[rand.IntN(n)].endpoint, nil
	}

	return nil, xerrors.ErrNotFoundEndpoint
}

// 轮询分配
func (r *GameRoute) roundRobinDispatch() (*endpoint.Endpoint, error) {
	if n := len(r.endpoints1); n > 0 {
		index := int(r.counter.Add(1) % uint64(n))

		return r.endpoints1[index].endpoint, nil
	}

	if n := len(r.endpoints2); n > 0 {
		index := int(r.counter.Add(1) % uint64(n))

		return r.endpoints2[index].endpoint, nil
	}

	return nil, xerrors.ErrNotFoundEndpoint
}

// 加权轮询分配
func (r *GameRoute) weightRoundRobinDispatch() (*endpoint.Endpoint, error) {
	var (
		selected    *serviceEndpoint
		totalWeight int
	)

	if len(r.endpoints1) > 0 {
		for i := range r.endpoints1 {
			se := r.endpoints1[i]
			se.currWeight += se.weight

			totalWeight += se.weight

			if selected == nil || se.currWeight > selected.currWeight {
				selected = se
			}
		}

		selected.currWeight -= totalWeight

		return selected.endpoint, nil
	}

	if len(r.endpoints2) > 0 {
		for i := range r.endpoints2 {
			se := r.endpoints2[i]
			se.currWeight += se.weight

			totalWeight += se.weight

			if selected == nil || se.currWeight > selected.currWeight {
				selected = se
			}
		}

		selected.currWeight -= totalWeight

		return selected.endpoint, nil
	}

	return nil, xerrors.ErrNotFoundEndpoint
}
