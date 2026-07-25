package main

import (
	eventbuscomponent "github.com/xbaseio/xbase/component/eventbus"
	gatepolicy "github.com/xbaseio/xbase/component/eventbus"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/bootstrap"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

func main() {
	defer func() { _ = xlog.Sync() }()

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
	xlog.Logger().Info("publish service status policy success",
		zap.Strings("gateNames", payload.GateNames),
		zap.Int("grayTrafficPercent", payload.Policy.GrayTrafficPercent),
		zap.Int("grayWhitelist", len(payload.Policy.GrayWhitelist)),
		zap.Int("testWhitelist", len(payload.Policy.TestWhitelist)),
	)
}
