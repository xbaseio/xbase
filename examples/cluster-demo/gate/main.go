package main

import (
	"github.com/xbaseio/xbase"
	"github.com/xbaseio/xbase/cluster/gate"
	eventbuscomponent "github.com/xbaseio/xbase/component/eventbus"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/bootstrap"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/observability"
)

func main() {
	locator := bootstrap.NewLocator()
	reg := bootstrap.NewRegistry()
	server := bootstrap.NewGateServer()
	debugServer := observability.NewDebugServer(
		etc.Get("etc.clusterDemo.debug.addr").String(),
		locator,
		reg,
	)

	g := gate.NewGate(
		gate.WithLocator(locator),
		gate.WithRegistry(reg),
		gate.WithServer(server),
	)
	globalEventbus := eventbuscomponent.NewGlobal(bootstrap.NewEventbus())

	container := xbase.NewContainer()
	container.Add(debugServer, globalEventbus, g)
	container.Serve()
}
