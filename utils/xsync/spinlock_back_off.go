package xsync

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type spinLockBackoff struct {
	state atomic.Uint32
	_     [60]byte
}

const (
	activeSpin  = 8
	activeCount = 16
)

func (sl *spinLockBackoff) Lock() {
	if sl.state.CompareAndSwap(0, 1) {
		return
	}

	for {
		for range activeSpin {
			for range activeCount {
				if sl.state.Load() == 0 && sl.state.CompareAndSwap(0, 1) {
					return
				}
			}
			runtime.Gosched()
		}
	}

}
func (sl *spinLockBackoff) Unlock() {
	sl.state.Store(0)
}
func NewSpinLockBackoff() sync.Locker {
	return new(spinLockBackoff)
}
