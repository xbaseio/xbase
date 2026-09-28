package node_test

import (
	"math"
	"sync/atomic"
	"testing"
)

var idx atomic.Uint64

func TestNewClient(t *testing.T) {
	idx.Add(math.MaxUint64)

	t.Log(idx.Add(1))
	t.Log(idx.Add(1))
}
