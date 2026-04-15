package xsync

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type spinLockBackoff struct {
	state uint32
	_     [60]byte
}

const (
	activeSpin  = 8
	activeCount = 16
)

func (sl *spinLockBackoff) Lock() {
	if atomic.CompareAndSwapUint32(&sl.state, 0, 1) {
		return
	}

	for {
		for i := 0; i < activeSpin; i++ {
			for j := 0; j < activeCount; j++ {
				if atomic.LoadUint32(&sl.state) == 0 && atomic.CompareAndSwapUint32(&sl.state, 0, 1) {
					return
				}
			}
			runtime.Gosched()
		}
	}

}
func (sl *spinLockBackoff) Unlock() {
	atomic.StoreUint32(&sl.state, 0)
}
func NewSpinLockBackoff() sync.Locker {
	return new(spinLockBackoff)
}
