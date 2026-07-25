package eventbuscomponent

import (
	"context"
	"slices"

	"github.com/xbaseio/xbase/component"
	"github.com/xbaseio/xbase/eventbus"
	"github.com/xbaseio/xbase/xlog"
)

const DefaultServiceStatusPolicyTopic = "cluster.gate.service_status_policy"

type ServiceStatusPolicy struct {
	GrayWhitelist      []int64 `json:"gray_whitelist"`
	GrayTrafficPercent int     `json:"gray_traffic_percent"`
	GrayTrafficSalt    string  `json:"gray_traffic_salt"`
	TestWhitelist      []int64 `json:"test_whitelist"`
}

type ServiceStatusPolicyEvent struct {
	GateIDs   []string            `json:"gate_ids,omitempty"`
	GateNames []string            `json:"gate_names,omitempty"`
	Policy    ServiceStatusPolicy `json:"policy"`
}

type Global struct {
	component.Base
	eb eventbus.Eventbus
}

func NewGlobal(eb eventbus.Eventbus) *Global {
	return &Global{eb: eb}
}

func (g *Global) Name() string {
	return "eventbus-global"
}

func (g *Global) Init() {
	if g.eb != nil {
		eventbus.SetEventbus(g.eb)
	}
}

func PublishServiceStatusPolicy(payload *ServiceStatusPolicyEvent) {
	if payload == nil {
		return
	}

	if err := eventbus.Publish(context.Background(), DefaultServiceStatusPolicyTopic, payload); err != nil {
		xlog.Sugar().Errorf("publish service status policy failed, payload=%#v err=%v", payload, err)
	}
}

func SubscribeServiceStatusPolicy(handler func(uuid string, payload *ServiceStatusPolicyEvent)) eventbus.EventHandler {
	wrapper := func(event *eventbus.Event) {
		if handler == nil || event == nil {
			return
		}

		payload := &ServiceStatusPolicyEvent{}
		if err := event.Payload.Scan(payload); err != nil {
			return
		}

		handler(event.ID, payload)
	}

	if err := eventbus.Subscribe(context.Background(), DefaultServiceStatusPolicyTopic, wrapper); err != nil {
		xlog.Sugar().Errorf("subscribe service status policy failed: %v", err)
	}

	return wrapper
}

func (e *ServiceStatusPolicyEvent) MatchGate(id, name string) bool {
	if e == nil {
		return false
	}

	if len(e.GateIDs) == 0 && len(e.GateNames) == 0 {
		return true
	}

	return slices.Contains(e.GateIDs, id) || slices.Contains(e.GateNames, name)
}
