package xsync

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type spinLockFast uint32

const (
	maxBackoff = 16
)

// Lock 获取锁（自旋 + 指数退避）。
func (sl *spinLockFast) Lock() {
	backoff := 1

	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		// 指数退避：减少 CPU 竞争
		for i := 0; i < backoff; i++ {
			runtime.Gosched()
		}
		if backoff < maxBackoff {
			backoff <<= 1
		}
	}
}

func (sl *spinLockFast) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

func NewSpinLockFast() sync.Locker {
	return new(spinLockFast)
}
