package main

import (
	eventbuscomponent "github.com/xbaseio/xbase/component/eventbus"
	gatepolicy "github.com/xbaseio/xbase/component/eventbus"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/bootstrap"
	"github.com/xbaseio/xbase/log"
)

func main() {
	eventbuscomponent.NewGlobal(bootstrap.NewEventbus()).Init()

	payload := gatepolicy.ServiceStatusPolicyEvent{
		GateNames: []string{etc.Get("etc.cluster.gate.name", "demo-gate").String()},
		Policy: gatepolicy.ServiceStatusPolicy{
			GrayWhitelist:      etc.Get("etc.cluster.gate.grayWhitelist").Int64s(),
			GrayTrafficPercent: etc.Get("etc.cluster.gate.grayTrafficPercent").Int(),
			GrayTrafficSalt:    etc.Get("etc.cluster.gate.grayTrafficSalt").String(),
			TestWhitelist:      etc.Get("etc.cluster.gate.testWhitelist").Int64s(),
		},
	}

	gatepolicy.PublishServiceStatusPolicy(&payload)

	log.Infof("publish service status policy success, gateNames=%v grayTrafficPercent=%d grayWhitelist=%d testWhitelist=%d",
		payload.GateNames, payload.Policy.GrayTrafficPercent, len(payload.Policy.GrayWhitelist), len(payload.Policy.TestWhitelist))
}
