//go:build !race

package xants_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/xbaseio/xbase/utils/xants"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024 字节
	MiB // 1048576 字节
)

const (
	Param    = 100   // 模拟任务参数
	AntsSize = 1000  // 协程池容量
	TestSize = 10000 // 测试规模
	n        = 100000
)

var curMem uint64

// TestAntsPoolWaitToGetWorker 测试：任务提交时等待 worker 可用。
func TestAntsPoolWaitToGetWorker(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := xants.NewPool(AntsSize)
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Submit(func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()

	t.Logf("pool, 当前运行 worker 数:%d", p.Running())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem

	t.Logf("内存使用:%d MB", curMem)
}

// TestAntsPoolWaitToGetWorkerPreMalloc 测试：开启预分配模式。
func TestAntsPoolWaitToGetWorkerPreMalloc(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := xants.NewPool(AntsSize, xants.WithPreAlloc(true))
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Submit(func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()

	t.Logf("pool, 当前运行 worker 数:%d", p.Running())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem

	t.Logf("内存使用:%d MB", curMem)
}

// TestAntsPoolWithFuncWaitToGetWorker 测试：带函数池（any）模式。
func TestAntsPoolWithFuncWaitToGetWorker(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := xants.NewPoolWithFunc(AntsSize, func(i any) {
		demoPoolFunc(i)
		wg.Done()
	})
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Invoke(Param)
	}
	wg.Wait()

	t.Logf("pool with func, 当前运行 worker 数:%d", p.Running())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem

	t.Logf("内存使用:%d MB", curMem)
}

// TestAntsPoolWithFuncGenericWaitToGetWorker 测试：泛型版本函数池。
func TestAntsPoolWithFuncGenericWaitToGetWorker(t *testing.T) {
	var wg sync.WaitGroup
	p, _ := xants.NewPoolWithFuncGeneric(AntsSize, func(i int) {
		demoPoolFuncInt(i)
		wg.Done()
	})
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Invoke(Param)
	}
	wg.Wait()

	t.Logf("pool with func, 当前运行 worker 数:%d", p.Running())

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem

	t.Logf("内存使用:%d MB", curMem)
}
