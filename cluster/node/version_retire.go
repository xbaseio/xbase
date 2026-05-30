package node

import (
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/cluster/versionretire"
)

func (n *Node) watchVersionRetire() {
	versionretire.Start(versionretire.Options{
		Ctx:         n.ctx,
		Registry:    n.opts.registry,
		ServiceName: cluster.Node.String(),
		Kind:        cluster.Node.String(),
		ID:          n.opts.id,
		Version:     n.opts.version,
		Alias:       n.opts.name,
		GameID:      n.opts.gameID,
		MatchGameID: true,
		RetireDelay: n.opts.retireDelay,
		Timeout:     n.opts.timeout,
		Shutdown: func() {
			n.Close()
			n.Destroy()
		},
	})
}
