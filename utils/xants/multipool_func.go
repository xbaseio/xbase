package xants

import (
	"context"
	"math"
	"sync/atomic"
	"time"
)

// MultiPoolWithFunc 由多个子协程池组成。
// 通过细粒度拆分池来降低锁竞争，从而提升整体性能。
//
// 适用于：
// - 任务量非常大
// - 单个协程池可能成为瓶颈的场景
type MultiPoolWithFunc struct {
	pools []*PoolWithFunc
	index uint32
	state int32
	lbs   LoadBalancingStrategy
}

// NewMultiPoolWithFunc 创建一个多池结构。
// 参数：
// - size：子池数量
// - sizePerPool：每个子池容量
// - fn：任务函数
// - lbs：负载均衡策略
func NewMultiPoolWithFunc(size, sizePerPool int, fn func(any), lbs LoadBalancingStrategy, options ...Option) (*MultiPoolWithFunc, error) {
	if size <= 0 {
		return nil, ErrInvalidMultiPoolSize
	}

	if lbs != RoundRobin && lbs != LeastTasks {
		return nil, ErrInvalidLoadBalancingStrategy
	}

	pools := make([]*PoolWithFunc, size)
	for i := 0; i < size; i++ {
		pool, err := NewPoolWithFunc(sizePerPool, fn, options...)
		if err != nil {
			return nil, err
		}
		pools[i] = pool
	}

	return &MultiPoolWithFunc{pools: pools, index: math.MaxUint32, lbs: lbs}, nil
}

// next 根据负载均衡策略选择一个子池索引
func (mp *MultiPoolWithFunc) next(lbs LoadBalancingStrategy) (idx int) {
	switch lbs {
	case RoundRobin:
		// 轮询策略
		return int(atomic.AddUint32(&mp.index, 1) % uint32(len(mp.pools)))

	case LeastTasks:
		// 最少任务策略：选择当前运行 worker 最少的池
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

// Invoke 按负载均衡策略选择子池并提交任务
func (mp *MultiPoolWithFunc) Invoke(args any) (err error) {
	if mp.IsClosed() {
		return ErrPoolClosed
	}

	// 正常路径
	if err = mp.pools[mp.next(mp.lbs)].Invoke(args); err == nil {
		return
	}

	// 如果轮询模式下某个池满了，退化为最少任务策略
	if err == ErrPoolOverload && mp.lbs == RoundRobin {
		return mp.pools[mp.next(LeastTasks)].Invoke(args)
	}

	return
}

// Running 返回所有子池当前运行中的 worker 总数
func (mp *MultiPoolWithFunc) Running() (n int) {
	for _, pool := range mp.pools {
		n += pool.Running()
	}
	return
}

// RunningByIndex 返回指定子池当前运行中的 worker 数
func (mp *MultiPoolWithFunc) RunningByIndex(idx int) (int, error) {
	if idx < 0 || idx >= len(mp.pools) {
		return -1, ErrInvalidPoolIndex
	}
	return mp.pools[idx].Running(), nil
}

// Free 返回所有子池当前空闲 worker 总数
func (mp *MultiPoolWithFunc) Free() (n int) {
	for _, pool := range mp.pools {
		n += pool.Free()
	}
	return
}

// FreeByIndex 返回指定子池当前空闲 worker 数
func (mp *MultiPoolWithFunc) FreeByIndex(idx int) (int, error) {
	if idx < 0 || idx >= len(mp.pools) {
		return -1, ErrInvalidPoolIndex
	}
	return mp.pools[idx].Free(), nil
}

// Waiting 返回所有子池当前等待中的任务总数
func (mp *MultiPoolWithFunc) Waiting() (n int) {
	for _, pool := range mp.pools {
		n += pool.Waiting()
	}
	return
}

// WaitingByIndex 返回指定子池当前等待中的任务数
func (mp *MultiPoolWithFunc) WaitingByIndex(idx int) (int, error) {
	if idx < 0 || idx >= len(mp.pools) {
		return -1, ErrInvalidPoolIndex
	}
	return mp.pools[idx].Waiting(), nil
}

// Cap 返回整个多池的总容量
func (mp *MultiPoolWithFunc) Cap() (n int) {
	for _, pool := range mp.pools {
		n += pool.Cap()
	}
	return
}

// Tune 调整每个子池的容量
//
// 注意：该方法调整的是每个子池容量，
// 不是整个多池的总容量
func (mp *MultiPoolWithFunc) Tune(size int) {
	for _, pool := range mp.pools {
		pool.Tune(size)
	}
}

// IsClosed 判断多池是否已关闭
func (mp *MultiPoolWithFunc) IsClosed() bool {
	return atomic.LoadInt32(&mp.state) == CLOSED
}

// ReleaseTimeout 带超时关闭多池
// 会等待所有子池关闭，直到超时
func (mp *MultiPoolWithFunc) ReleaseTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return mp.ReleaseContext(ctx)
}

// ReleaseContext 带 context 关闭多池
// 会等待所有子池关闭，直到 context 结束
func (mp *MultiPoolWithFunc) ReleaseContext(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&mp.state, OPENED, CLOSED) {
		return ErrPoolClosed
	}

	pools := make([]contextReleaser, len(mp.pools))
	for i, p := range mp.pools {
		pools[i] = p
	}
	return releasePools(ctx, pools)
}

// Reboot 重启一个已关闭的多池
func (mp *MultiPoolWithFunc) Reboot() {
	if atomic.CompareAndSwapInt32(&mp.state, CLOSED, OPENED) {
		atomic.StoreUint32(&mp.index, 0)
		for _, pool := range mp.pools {
			pool.Reboot()
		}
	}
}
