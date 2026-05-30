package mesh

import (
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/cluster/versionretire"
)

func (m *Mesh) watchVersionRetire() {
	versionretire.Start(versionretire.Options{
		Ctx:         m.ctx,
		Registry:    m.opts.registry,
		ServiceName: cluster.Mesh.String(),
		Kind:        cluster.Mesh.String(),
		ID:          m.opts.id,
		Version:     m.opts.version,
		Alias:       m.opts.name,
		RetireDelay: m.opts.retireDelay,
		Timeout:     m.opts.timeout,
		Shutdown: func() {
			m.Close()
			m.Destroy()
		},
	})
}
