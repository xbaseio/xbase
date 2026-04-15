package node_test

import (
	"context"
	"testing"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/buffer"
	"github.com/xbaseio/xbase/internal/transporter/node"
	"github.com/xbaseio/xbase/utils/xuuid"
)

func TestBuilder(t *testing.T) {
	builder := node.NewBuilder(&node.Options{
		InsID:   xuuid.UUID(),
		InsKind: cluster.Gate,
	})

	client, err := builder.Build("127.0.0.1:49898")
	if err != nil {
		t.Fatal(err)
	}

	err = client.Deliver(context.Background(), 1, 2, buffer.NewNocopyBuffer([]byte("hello world")))
	if err != nil {
		t.Fatal(err)
	}
}
