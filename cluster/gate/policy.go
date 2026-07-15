package gate

import (
	gatepolicy "github.com/xbaseio/xbase/component/eventbus"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/registry"
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
	log.Infof("gate service status policy updated, gateID: %s gateName: %s grayTrafficPercent: %d grayWhitelist: %d testWhitelist: %d",
		g.opts.id, g.opts.name, current.GrayTrafficPercent, len(current.GrayWhitelist), len(current.TestWhitelist))

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
		log.Infof("gate service status policy event applied, eventID: %s gateID: %s gateName: %s grayTrafficPercent: %d grayWhitelist: %d testWhitelist: %d",
			uuid, g.opts.id, g.opts.name, current.GrayTrafficPercent, len(current.GrayWhitelist), len(current.TestWhitelist))
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
