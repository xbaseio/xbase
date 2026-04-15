// Package xringbuffer 实现一个对 GC 更友好的 xring buffer 对象池。
package xringbuffer

import (
	"math/bits"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/xbaseio/xbase/utils/xbuffer/xring"
)

const (
	minBitSize = 6  // 2**6 = 64，通常等于一个 CPU cache line 大小
	steps      = 20 // 分桶数量

	minSize = 1 << minBitSize

	calibrateCallsThreshold = 42000
	maxPercentile           = 0.95
)

// RingBuffer 是 xring.XBuffer 的类型别名。
type RingBuffer = xring.XBuffer

// Pool 表示 xring buffer 对象池。
//
// 不同类型、不同大小分布的 buffer 可以使用不同的池，
// 这样有助于减少内存浪费。
type Pool struct {
	calls       [steps]uint64
	calibrating uint64

	defaultSize uint64
	maxSize     uint64

	pool sync.Pool
}

var builtinPool Pool

// Get 从默认对象池中获取一个 RingBuffer。
func Get() *RingBuffer { return builtinPool.Get() }

// Get 从对象池中获取一个 RingBuffer。
//
// 获取到的 RingBuffer 使用完后可通过 Put 放回池中，
// 以减少内存分配和 GC 压力。
func (p *Pool) Get() *RingBuffer {
	v := p.pool.Get()
	if v != nil {
		return v.(*RingBuffer)
	}
	return xring.New(int(atomic.LoadUint64(&p.defaultSize)))
}

// Put 将 RingBuffer 放回默认对象池。
func Put(b *RingBuffer) { builtinPool.Put(b) }

// Put 将通过 Get 获取到的 RingBuffer 放回对象池。
//
// 放回池后不得再访问该对象，否则会产生数据竞争。
func (p *Pool) Put(b *RingBuffer) {
	idx := index(b.Len())

	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	maxSize := int(atomic.LoadUint64(&p.maxSize))
	if maxSize == 0 || b.Cap() <= maxSize {
		b.Reset()
		p.pool.Put(b)
	}
}

func (p *Pool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	a := make(callSizes, 0, steps)
	var callsSum uint64

	for i := uint64(0); i < steps; i++ {
		calls := atomic.SwapUint64(&p.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,
			size:  minSize << i,
		})
	}

	sort.Sort(a)

	defaultSize := a[0].size
	maxSize := defaultSize

	maxSum := uint64(float64(callsSum) * maxPercentile)
	callsSum = 0

	for i := 0; i < steps; i++ {
		if callsSum > maxSum {
			break
		}

		callsSum += a[i].calls
		size := a[i].size
		if size > maxSize {
			maxSize = size
		}
	}

	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)
	atomic.StoreUint64(&p.calibrating, 0)
}

type callSize struct {
	calls uint64
	size  uint64
}

type callSizes []callSize

func (cs callSizes) Len() int {
	return len(cs)
}

func (cs callSizes) Less(i, j int) bool {
	return cs[i].calls > cs[j].calls
}

func (cs callSizes) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

func index(n int) int {
	n--
	n >>= minBitSize

	idx := 0
	if n > 0 {
		idx = bits.Len(uint(n))
	}
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}
