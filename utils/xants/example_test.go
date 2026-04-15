//go:build !race

package xants_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/utils/xants"
)

var (
	sum int32
	wg  sync.WaitGroup
)

// incSum 包装函数（any 参数版本）
func incSum(i any) {
	incSumInt(i.(int32))
}

// incSumInt 实际执行逻辑：累加并通知完成
func incSumInt(i int32) {
	atomic.AddInt32(&sum, i)
	wg.Done()
}

// ExamplePool 示例：使用默认协程池与自定义协程池
func ExamplePool() {
	xants.Reboot() // 确保默认池可用

	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)

	// 使用默认协程池
	for i := 0; i < runTimes; i++ {
		j := i
		_ = xants.Submit(func() {
			incSumInt(int32(j))
		})
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	atomic.StoreInt32(&sum, 0)
	wg.Add(runTimes)

	// 使用自定义协程池
	pool, _ := xants.NewPool(10)
	defer pool.Release()

	for i := 0; i < runTimes; i++ {
		j := i
		_ = pool.Submit(func() {
			incSumInt(int32(j))
		})
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	// Output:
	// 结果为 499500
	// 结果为 499500
}

// ExamplePoolWithFunc 示例：使用带函数的协程池（any 参数）
func ExamplePoolWithFunc() {
	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)

	pool, _ := xants.NewPoolWithFunc(10, incSum)
	defer pool.Release()

	for i := 0; i < runTimes; i++ {
		_ = pool.Invoke(int32(i))
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	// Output: 结果为 499500
}

// ExamplePoolWithFuncGeneric 示例：使用泛型函数协程池
func ExamplePoolWithFuncGeneric() {
	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)

	pool, _ := xants.NewPoolWithFuncGeneric(10, incSumInt)
	defer pool.Release()

	for i := 0; i < runTimes; i++ {
		_ = pool.Invoke(int32(i))
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	// Output: 结果为 499500
}

// ExampleMultiPool 示例：使用多池（分片）协程池
func ExampleMultiPool() {
	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)

	mp, _ := xants.NewMultiPool(10, runTimes/10, xants.RoundRobin)
	defer mp.ReleaseTimeout(time.Second) // nolint:errcheck

	for i := 0; i < runTimes; i++ {
		j := i
		_ = mp.Submit(func() {
			incSumInt(int32(j))
		})
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	// Output: 结果为 499500
}

// ExampleMultiPoolWithFunc 示例：多池 + 函数版本（any）
func ExampleMultiPoolWithFunc() {
	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)

	mp, _ := xants.NewMultiPoolWithFunc(10, runTimes/10, incSum, xants.RoundRobin)
	defer mp.ReleaseTimeout(time.Second) // nolint:errcheck

	for i := 0; i < runTimes; i++ {
		_ = mp.Invoke(int32(i))
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	// Output: 结果为 499500
}

// ExampleMultiPoolWithFuncGeneric 示例：多池 + 泛型函数版本
func ExampleMultiPoolWithFuncGeneric() {
	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)

	mp, _ := xants.NewMultiPoolWithFuncGeneric(10, runTimes/10, incSumInt, xants.RoundRobin)
	defer mp.ReleaseTimeout(time.Second) // nolint:errcheck

	for i := 0; i < runTimes; i++ {
		_ = mp.Invoke(int32(i))
	}
	wg.Wait()

	fmt.Printf("结果为 %d\n", sum)

	// Output: 结果为 499500
}
