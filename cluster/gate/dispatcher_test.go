package gate

import (
	"context"
	"testing"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/packet"
)

func TestDispatchGateMessageCallsUpperDispatcher(t *testing.T) {
	var got *packet.Message
	p := &proxy{gate: &Gate{opts: &options{
		messageDispatcher: func(ctx Context) {
			got = ctx.Message()
		},
	}}}
	want := &packet.Message{GameID: cluster.GateGameID, MessageID: 2000}

	p.dispatchGateMessage(context.Background(), nil, want)

	if got != want {
		t.Fatalf("dispatcher message = %p, want %p", got, want)
	}
}
