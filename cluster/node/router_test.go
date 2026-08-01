package node

import (
	"maps"
	"sync/atomic"
	"testing"

	"github.com/xbaseio/xbase/cluster"
)

func TestRequestMetadata(t *testing.T) {
	n := NewNode()
	req := n.reqPool.Get().(*request)
	req.metadata = map[string]string{"agentCode": "agent001", "roomID": "10001"}

	metadata := req.Metadata()
	if !maps.Equal(metadata, req.metadata) {
		t.Fatalf("metadata = %#v, want %#v", metadata, req.metadata)
	}
	metadata["agentCode"] = "changed"
	if req.Metadata()["agentCode"] != "agent001" {
		t.Fatal("request metadata was mutated by the caller")
	}
}

func TestRouterMessageDispatcherReceivesUndeclaredMessages(t *testing.T) {
	n := NewNode()
	var calls atomic.Int32
	n.Proxy().BindMessageDispatcher(func(ctx Context) {
		if ctx.MessageID() != 1001 {
			t.Errorf("unexpected message id: %d", ctx.MessageID())
		}
		calls.Add(1)
	})

	req := n.reqPool.Get().(*request)
	req.message.MessageID = 1001
	n.router.handle(req)

	if got := calls.Load(); got != 1 {
		t.Fatalf("dispatcher calls = %d, want 1", got)
	}
}

func TestRouterMessageDispatcherUsesDeclaredRouteMiddleware(t *testing.T) {
	n := NewNode()
	var calls atomic.Int32
	var middlewareCalls atomic.Int32
	n.Proxy().BindMessageDispatcher(func(Context) { calls.Add(1) })
	n.Proxy().SetRoutePolicy(1001, RoutePolicy{
		Middlewares: []MiddlewareHandler{
			func(m *Middleware, ctx Context) {
				middlewareCalls.Add(1)
				m.Next(ctx)
			},
		},
	})

	req := n.reqPool.Get().(*request)
	req.message.MessageID = 1001
	n.router.handle(req)

	if got := middlewareCalls.Load(); got != 1 {
		t.Fatalf("middleware calls = %d, want 1", got)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("dispatcher calls = %d, want 1", got)
	}
}

func TestRouterMessageDispatcherCanDeclareStatefulMetadata(t *testing.T) {
	n := NewNode()
	n.Proxy().BindMessageDispatcher(func(Context) {})
	n.Proxy().SetRoutePolicy(1001, StatefulPolicy)

	stateful, ok := n.router.CheckRouteStateful(1001)
	if !ok || !stateful {
		t.Fatalf("stateful metadata = (%t, %t), want (true, true)", stateful, ok)
	}

	if n.getState() != cluster.Shut {
		t.Fatalf("new node state = %v, want shut", n.getState())
	}
}
