package xants

import (
	"context"
	"math"
	"sync/atomic"
	"time"
)

// MultiPoolWithFuncGeneric 是 MultiPoolWithFunc 的泛型版本。
type MultiPoolWithFuncGeneric[T any] struct {
	pools []*PoolWithFuncGeneric[T]
	index uint32
	state int32
	lbs   LoadBalancingStrategy
}

// NewMultiPoolWithFuncGeneric 创建一个泛型多池。
// 参数分别为：池数量、每个池的容量、任务函数、负载均衡策略。
func NewMultiPoolWithFuncGeneric[T any](size, sizePerPool int, fn func(T), lbs LoadBalancingStrategy, options ...Option) (*MultiPoolWithFuncGeneric[T], error) {
	if size <= 0 {
		return nil, ErrInvalidMultiPoolSize
	}

	if lbs != RoundRobin && lbs != LeastTasks {
		return nil, ErrInvalidLoadBalancingStrategy
	}

	pools := make([]*PoolWithFuncGeneric[T], size)
	for i := 0; i < size; i++ {
		pool, err := NewPoolWithFuncGeneric(sizePerPool, fn, options...)
		if err != nil {
			return nil, err
		}
		pools[i] = pool
	}

	return &MultiPoolWithFuncGeneric[T]{pools: pools, index: math.MaxUint32, lbs: lbs}, nil
}

func (mp *MultiPoolWithFuncGeneric[T]) next(lbs LoadBalancingStrategy) (idx int) {
	switch lbs {
	case RoundRobin:
		return int(atomic.AddUint32(&mp.index, 1) % uint32(len(mp.pools)))

	case LeastTasks:
		leastTasks := 1<<31 - 1
		for i, pool := range mp.pools {
			if n := pool.Running(); n < leastTasks {
				leastTasks = n
				idx = i
			}
		}
		return
	}

	return -1
}

// Invoke 按负载均衡策略选择一个池并提交任务。
func (mp *MultiPoolWithFuncGeneric[T]) Invoke(args T) (err error) {
	if mp.IsClosed() {
		return ErrPoolClosed
	}

	if err = mp.pools[mp.next(mp.lbs)].Invoke(args); err == nil {
		return
	}

	if err == ErrPoolOverload && mp.lbs == RoundRobin {
		return mp.pools[mp.next(LeastTasks)].Invoke(args)
	}

	return
}

// Running 返回所有池当前运行中的 worker 总数。
func (mp *MultiPoolWithFuncGeneric[T]) Running() (n int) {
	for _, pool := range mp.pools {
		n += pool.Running()
	}
	return
}

// RunningByIndex 返回指定池当前运行中的 worker 数。
func (mp *MultiPoolWithFuncGeneric[T]) RunningByIndex(idx int) (int, error) {
	if idx < 0 || idx >= len(mp.pools) {
		return -1, ErrInvalidPoolIndex
	}
	return mp.pools[idx].Running(), nil
}

// Free 返回所有池当前空闲 worker 总数。
func (mp *MultiPoolWithFuncGeneric[T]) Free() (n int) {
	for _, pool := range mp.pools {
		n += pool.Free()
	}
	return
}

// FreeByIndex 返回指定池当前空闲 worker 数。
func (mp *MultiPoolWithFuncGeneric[T]) FreeByIndex(idx int) (int, error) {
	if idx < 0 || idx >= len(mp.pools) {
		return -1, ErrInvalidPoolIndex
	}
	return mp.pools[idx].Free(), nil
}

// Waiting 返回所有池当前等待中的任务总数。
func (mp *MultiPoolWithFuncGeneric[T]) Waiting() (n int) {
	for _, pool := range mp.pools {
		n += pool.Waiting()
	}
	return
}

// WaitingByIndex 返回指定池当前等待中的任务数。
func (mp *MultiPoolWithFuncGeneric[T]) WaitingByIndex(idx int) (int, error) {
	if idx < 0 || idx >= len(mp.pools) {
		return -1, ErrInvalidPoolIndex
	}
	return mp.pools[idx].Waiting(), nil
}

// Cap 返回整个多池的总容量。
func (mp *MultiPoolWithFuncGeneric[T]) Cap() (n int) {
	for _, pool := range mp.pools {
		n += pool.Cap()
	}
	return
}

// Tune 调整多池中每个池的容量。
//
// 注意：该方法调整的是每个子池容量，
// 不是整个多池的总容量。
func (mp *MultiPoolWithFuncGeneric[T]) Tune(size int) {
	for _, pool := range mp.pools {
		pool.Tune(size)
	}
}

// IsClosed 返回多池是否已关闭。
func (mp *MultiPoolWithFuncGeneric[T]) IsClosed() bool {
	return atomic.LoadInt32(&mp.state) == CLOSED
}

// ReleaseTimeout 带超时关闭多池。
// 它会等待所有子池关闭，直到超时。
func (mp *MultiPoolWithFuncGeneric[T]) ReleaseTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return mp.ReleaseContext(ctx)
}

// ReleaseContext 带 context 关闭多池。
// 它会等待所有子池关闭，直到 context 结束。
func (mp *MultiPoolWithFuncGeneric[T]) ReleaseContext(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&mp.state, OPENED, CLOSED) {
		return ErrPoolClosed
	}

	pools := make([]contextReleaser, len(mp.pools))
	for i, p := range mp.pools {
		pools[i] = p
	}
	return releasePools(ctx, pools)
}

// Reboot 重启一个已释放的多池。
func (mp *MultiPoolWithFuncGeneric[T]) Reboot() {
	if atomic.CompareAndSwapInt32(&mp.state, CLOSED, OPENED) {
		atomic.StoreUint32(&mp.index, 0)
		for _, pool := range mp.pools {
			pool.Reboot()
		}
	}
}
