package lock_test

import (
	"context"
	"testing"

	"github.com/xbaseio/xbase/lock"
)

func TestMake(t *testing.T) {
	locker := lock.Make("lockName")

	if err := locker.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}

	defer locker.Release(context.Background())

}
