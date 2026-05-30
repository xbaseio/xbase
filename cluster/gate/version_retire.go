package gate

import (
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/cluster/versionretire"
)

func (g *Gate) watchVersionRetire() {
	versionretire.Start(versionretire.Options{
		Ctx:         g.ctx,
		Registry:    g.opts.registry,
		ServiceName: cluster.Gate.String(),
		Kind:        cluster.Gate.String(),
		ID:          g.opts.id,
		Version:     g.opts.version,
		Alias:       g.opts.name,
		RetireDelay: g.opts.retireDelay,
		Timeout:     g.opts.timeout,
		Shutdown: func() {
			g.Close()
			g.Destroy()
		},
	})
}
