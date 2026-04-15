package pool_test

import (
	"testing"

	"github.com/xbaseio/xbase/core/pool"
)

func TestFactory_Get(t *testing.T) {
	factory := pool.NewFactory(func(name string) (*Instance, error) {
		return &Instance{Name: name}, nil
	})

	ins, err := factory.Get("mysql")
	if err != nil {
		t.Fatalf("get instance failed: %v", err)
	}

	t.Logf("instance name: %v", ins.Name)
}

type Instance struct {
	Name string
}
