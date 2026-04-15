//go:build !race

package xants_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/xbaseio/xbase/utils/xants"
)

const (
	RunTimes           = 1e6              // 每轮执行任务数量
	PoolCap            = 5e4              // 协程池容量
	BenchParam         = 10               // 模拟任务耗时参数（毫秒）
	DefaultExpiredTime = 10 * time.Second // worker 过期时间
)

// demoFunc 模拟一个简单任务（固定耗时）。
func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}

// demoPoolFunc 模拟带参数的任务（any 版本）。
func demoPoolFunc(args any) {
	n := args.(int)
	time.Sleep(time.Duration(n) * time.Millisecond)
}

// demoPoolFuncInt 模拟带参数任务（强类型版本）。
func demoPoolFuncInt(n int) {
	time.Sleep(time.Duration(n) * time.Millisecond)
}

var stopLongRunningFunc int32

// longRunningFunc 模拟长时间运行的任务（自旋 + 主动让出 CPU）。
func longRunningFunc() {
	for atomic.LoadInt32(&stopLongRunningFunc) == 0 {
		runtime.Gosched()
	}
}

// longRunningPoolFunc 模拟通过 channel 控制结束的长任务（any）。
func longRunningPoolFunc(arg any) {
	<-arg.(chan struct{})
}

// longRunningPoolFuncCh 模拟通过 channel 控制结束的长任务（强类型）。
func longRunningPoolFuncCh(ch chan struct{}) {
	<-ch
}

// BenchmarkGoroutines 基准：直接使用 goroutine。
func BenchmarkGoroutines(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				demoFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// BenchmarkChannel 基准：使用 channel 作为信号量控制并发。
func BenchmarkChannel(b *testing.B) {
	var wg sync.WaitGroup
	sema := make(chan struct{}, PoolCap)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				demoFunc()
				<-sema
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// BenchmarkErrGroup 基准：使用 errgroup 控制并发数量。
func BenchmarkErrGroup(b *testing.B) {
	var wg sync.WaitGroup
	var pool errgroup.Group
	pool.SetLimit(PoolCap)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			pool.Go(func() error {
				demoFunc()
				wg.Done()
				return nil
			})
		}
		wg.Wait()
	}
}

// BenchmarkAntsPool 基准：使用单协程池（xants）。
func BenchmarkAntsPool(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := xants.NewPool(PoolCap, xants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

// BenchmarkAntsMultiPool 基准：使用多池（分片）协程池。
func BenchmarkAntsMultiPool(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := xants.NewMultiPool(10, PoolCap/10, xants.RoundRobin, xants.WithExpiryDuration(DefaultExpiredTime))
	defer p.ReleaseTimeout(DefaultExpiredTime) //nolint:errcheck

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

// BenchmarkGoroutinesThroughput 基准：直接 goroutine 的吞吐测试（不等待完成）。
func BenchmarkGoroutinesThroughput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			go demoFunc()
		}
	}
}

// BenchmarkSemaphoreThroughput 基准：信号量控制下的吞吐测试。
func BenchmarkSemaphoreThroughput(b *testing.B) {
	sema := make(chan struct{}, PoolCap)
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				demoFunc()
				<-sema
			}()
		}
	}
}

// BenchmarkAntsPoolThroughput 基准：单池协程池吞吐测试。
func BenchmarkAntsPoolThroughput(b *testing.B) {
	p, _ := xants.NewPool(PoolCap, xants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(demoFunc)
		}
	}
}

// BenchmarkAntsMultiPoolThroughput 基准：多池协程池吞吐测试。
func BenchmarkAntsMultiPoolThroughput(b *testing.B) {
	p, _ := xants.NewMultiPool(10, PoolCap/10, xants.RoundRobin, xants.WithExpiryDuration(DefaultExpiredTime))
	defer p.ReleaseTimeout(DefaultExpiredTime) //nolint:errcheck

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(demoFunc)
		}
	}
}

// BenchmarkParallelAntsPoolThroughput 基准：并行模式下单池吞吐测试。
func BenchmarkParallelAntsPoolThroughput(b *testing.B) {
	p, _ := xants.NewPool(PoolCap, xants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = p.Submit(demoFunc)
		}
	})
}

// BenchmarkParallelAntsMultiPoolThroughput 基准：并行模式下多池吞吐测试。
func BenchmarkParallelAntsMultiPoolThroughput(b *testing.B) {
	p, _ := xants.NewMultiPool(10, PoolCap/10, xants.RoundRobin, xants.WithExpiryDuration(DefaultExpiredTime))
	defer p.ReleaseTimeout(DefaultExpiredTime) //nolint:errcheck

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = p.Submit(demoFunc)
		}
	})
}
