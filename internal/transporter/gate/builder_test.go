package gate_test

import (
	"context"
	"testing"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/internal/transporter/gate"
	"github.com/xbaseio/xbase/session"
	"github.com/xbaseio/xbase/utils/xuuid"
)

func TestBuilder(t *testing.T) {
	builder := gate.NewBuilder(&gate.Options{
		InsID:   xuuid.UUID(),
		InsKind: cluster.Node,
	})

	client, err := builder.Build("127.0.0.1:49899")
	if err != nil {
		t.Fatal(err)
	}

	ip, err := client.GetIP(context.Background(), session.User, 1)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ip: %v", ip)

	ip, err = client.GetIP(context.Background(), session.User, 1)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ip: %v", ip)
}

func TestBuilder_Fault(t *testing.T) {
	builder := gate.NewBuilder(&gate.Options{
		InsID:   xuuid.UUID(),
		InsKind: cluster.Node,
	})

	for i := range 3 {
		if _, err := builder.Build("127.0.0.1:49899"); err != nil {
			t.Log(err)
			time.Sleep(time.Duration(i+1) * time.Second)
		} else {
			t.Log("build success")
		}
	}
}
