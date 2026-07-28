package gate

import (
	gatepolicy "github.com/xbaseio/xbase/component/eventbus"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

func (g *Gate) ServiceStatusPolicy() gatepolicy.ServiceStatusPolicy {
	if g == nil || g.opts == nil {
		return gatepolicy.ServiceStatusPolicy{}
	}

	return g.opts.serviceStatusPolicy()
}

func (g *Gate) ApplyServiceStatusPolicy(policy gatepolicy.ServiceStatusPolicy, isMathchGate bool) gatepolicy.ServiceStatusPolicy {
	if g == nil || g.opts == nil {
		return gatepolicy.ServiceStatusPolicy{}
	}

	current := g.opts.updateServiceStatusPolicy(policy, isMathchGate)
	xlog.Logger().Info("gate service status policy updated, gateID: gateName: grayTrafficPercent: grayWhitelist: testWhitelist", zap.Any("id", g.opts.id), zap.Any("name", g.opts.name), zap.Any("grayTrafficPercent", current.GrayTrafficPercent), zap.Any("arg4", len(current.GrayWhitelist)), zap.Any("arg5", len(current.TestWhitelist)))

	return current
}

func (g *Gate) subscribeServiceStatusPolicy() {
	if g == nil || g.opts == nil {
		return
	}

	gatepolicy.SubscribeServiceStatusPolicy(func(uuid string, payload *gatepolicy.ServiceStatusPolicyEvent) {
		if payload == nil {
			return
		}
		isMathchGate := payload.MatchGate(g.opts.id, g.opts.name)

		current := g.ApplyServiceStatusPolicy(payload.Policy, isMathchGate)
		xlog.Logger().Info("gate service status policy event applied, eventID: gateID: gateName: grayTrafficPercent: grayWhitelist: testWhitelist", zap.Any("uuid", uuid), zap.Any("id", g.opts.id), zap.Any("name", g.opts.name), zap.Any("grayTrafficPercent", current.GrayTrafficPercent), zap.Any("arg5", len(current.GrayWhitelist)), zap.Any("arg6", len(current.TestWhitelist)))
	})
}

func (g *Gate) resolveServiceStatus(uid int64) registry.ServiceStatus {
	return g.resolveServiceStatusDecision(uid).targetStatus
}

func (g *Gate) resolveServiceStatusDecision(uid int64) serviceStatusDecision {
	if g == nil || g.opts == nil {
		return serviceStatusDecision{}
	}

	return g.opts.resolveServiceStatusDecision(uid)
}
