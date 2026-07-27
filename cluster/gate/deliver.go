package gate

import (
	"context"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/internal/link"
	"github.com/xbaseio/xbase/xerrors"
)

// Deliver sends an application message to the node currently bound to uid for
// the message's game route.
func (g *Gate) Deliver(ctx context.Context, cid, uid int64, message *cluster.Message) error {
	if g == nil || uid <= 0 || message == nil || message.GameID <= cluster.GateGameID || message.MessageID <= 0 {
		return xerrors.ErrInvalidArgument
	}
	if ctx == nil {
		ctx = g.ctx
	}

	return g.proxy.nodeLinker.Deliver(ctx, &link.DeliverArgs{
		CID:       cid,
		UID:       uid,
		GameID:    message.GameID,
		MessageID: message.MessageID,
		Buffer:    message,
	})
}
