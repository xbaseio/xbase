package main

import (
	"github.com/xbaseio/xbase"
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/cluster/mesh"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/bootstrap"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/observability"
)

func main() {
	reg := bootstrap.NewRegistry()
	transporter := bootstrap.NewTransporter()
	debugServer := observability.NewDebugServer(
		etc.Get("etc.clusterDemo.debug.addr").String(),
		nil,
		reg,
	)

	m := mesh.NewMesh(
		mesh.WithRegistry(reg),
		mesh.WithTransporter(transporter),
	)

	m.Proxy().AddHookListener(cluster.Start, func(proxy *mesh.Proxy) {
		observability.Info("mesh_started", "id", proxy.GetID(), "name", proxy.GetName())
	})

	container := xbase.NewContainer()
	container.Add(debugServer, m)
	container.Serve()
}
