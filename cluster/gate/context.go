package gate

import (
	"context"

	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
)

// Context is the upper-layer context for a Gate control message.
type Context interface {
	Context() context.Context
	Conn() network.Conn
	Message() *packet.Message
	CID() int64
	UID() int64
	Bind(uid int64) error
}

type messageContext struct {
	ctx     context.Context
	gate    *Gate
	conn    network.Conn
	message *packet.Message
}

func (c *messageContext) Context() context.Context { return c.ctx }
func (c *messageContext) Conn() network.Conn       { return c.conn }
func (c *messageContext) Message() *packet.Message { return c.message }
func (c *messageContext) CID() int64               { return c.conn.ID() }
func (c *messageContext) UID() int64               { return c.conn.UID() }

// Bind initializes the current connection after upper-layer login validation
// succeeds. When a lobby route (GameID = 1) exists, it also binds the user to
// a lobby node.
func (c *messageContext) Bind(uid int64) error {
	return c.gate.bind(c.ctx, c.conn.ID(), uid, true)
}
