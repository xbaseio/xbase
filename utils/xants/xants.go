// 使用 ants，Go 应用可以限制活跃 goroutine 的数量，
// 高效复用 goroutine，并显著降低内存占用。
// xants 特别适用于频繁创建和销毁大量 goroutine 的场景，
// 如高并发批处理系统、HTTP 服务、异步任务处理等。
package xants

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/utils/xsync"
)

const (
	// DefaultAntsPoolSize 默认协程池容量
	DefaultAntsPoolSize = math.MaxInt32

	// DefaultCleanIntervalTime 默认清理 goroutine 的时间间隔
	DefaultCleanIntervalTime = time.Second
)

const (
	// OPENED 表示池处于开启状态
	OPENED = iota

	// CLOSED 表示池已关闭
	CLOSED
)

var (
	// 未提供任务函数时返回
	ErrLackPoolFunc = errors.New("must provide function for pool")

	// 设置负数过期时间时返回
	ErrInvalidPoolExpiry = errors.New("invalid expiry for pool")

	// 向已关闭池提交任务时返回
	ErrPoolClosed = errors.New("this pool has been closed")

	// 池已满且无可用 worker 时返回
	ErrPoolOverload = errors.New("too many goroutines blocked on submit or Nonblocking is set")

	// 预分配模式下设置非法容量时返回
	ErrInvalidPreAllocSize = errors.New("can not set up a negative capacity under PreAlloc mode")

	// 操作超时时返回
	ErrTimeout = errors.New("operation timed out")

	// 获取非法 pool 索引时返回
	ErrInvalidPoolIndex = errors.New("invalid pool index")

	// 创建 MultiPool 时负载均衡策略非法
	ErrInvalidLoadBalancingStrategy = errors.New("invalid load-balancing strategy")

	// 创建 MultiPool 时 size 非法
	ErrInvalidMultiPoolSize = errors.New("invalid size for multiple pool")

	// workerChanCap 决定 worker 的 channel 是否为带缓冲
	// 用于获得最佳性能，灵感来源 fasthttp
	workerChanCap = func() int {
		// 单核：使用阻塞 channel（减少上下文切换）
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}

		// 多核：使用非阻塞 channel（避免 sender 被拖慢）
		return 1
	}()

	defaultLogger = Logger(log.New(os.Stderr, "[xants]: ", log.LstdFlags|log.Lmsgprefix|log.Lmicroseconds))

	// 初始化默认协程池
	defaultAntsPool, _ = NewPool(DefaultAntsPoolSize)
)

// Submit 提交任务到默认池
func Submit(task func()) error {
	return defaultAntsPool.Submit(task)
}

// Running 返回当前运行中的 goroutine 数量
func Running() int {
	return defaultAntsPool.Running()
}

// Cap 返回默认池容量
func Cap() int {
	return defaultAntsPool.Cap()
}

// Free 返回可用 worker 数量
func Free() int {
	return defaultAntsPool.Free()
}

// Release 关闭默认池
func Release() {
	defaultAntsPool.Release()
}

// ReleaseTimeout 带超时关闭
func ReleaseTimeout(timeout time.Duration) error {
	return defaultAntsPool.ReleaseTimeout(timeout)
}

// ReleaseContext 带 context 关闭
// 如果 ctx 为 nil，则等同于 Release（立即返回）
func ReleaseContext(ctx context.Context) error {
	return defaultAntsPool.ReleaseContext(ctx)
}

// Reboot 重启默认池
func Reboot() {
	defaultAntsPool.Reboot()
}

// Logger 用于日志输出
type Logger interface {
	// Printf 语义与 log.Printf 一致
	Printf(format string, args ...any)
}

// poolCommon 包含所有 Pool 的核心字段
type poolCommon struct {
	// 协程池容量，负数表示无限容量
	capacity int32

	// 当前运行中的 goroutine 数量
	running int32

	// 保护 worker 队列的锁
	lock sync.Locker

	// worker 队列
	workers workerQueue

	// 状态（OPENED / CLOSED）
	state int32

	// 条件变量，用于等待空闲 worker
	cond *sync.Cond

	// 所有 worker 完成信号
	allDone chan struct{}

	// 确保只关闭一次
	once *sync.Once

	// worker 缓存（加速获取）
	workerCache sync.Pool

	// 当前阻塞的 goroutine 数量
	waiting int32

	purgeDone int32
	purgeCtx  context.Context
	stopPurge context.CancelFunc

	ticktockDone int32
	ticktockCtx  context.Context
	stopTicktock context.CancelFunc

	now int64

	options *Options
}

func newPool(size int, options ...Option) (*poolCommon, error) {
	if size <= 0 {
		size = -1
	}

	opts := loadOptions(options...)

	if !opts.DisablePurge {
		if expiry := opts.ExpiryDuration; expiry < 0 {
			return nil, ErrInvalidPoolExpiry
		} else if expiry == 0 {
			opts.ExpiryDuration = DefaultCleanIntervalTime
		}
	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}

	p := &poolCommon{
		capacity: int32(size),
		allDone:  make(chan struct{}),
		lock:     xsync.NewSpinLockFast(),
		once:     &sync.Once{},
		options:  opts,
	}
	if p.options.PreAlloc {
		if size == -1 {
			return nil, ErrInvalidPreAllocSize
		}
		p.workers = newWorkerQueue(queueTypeLoopQueue, size)
	} else {
		p.workers = newWorkerQueue(queueTypeStack, 0)
	}

	p.cond = sync.NewCond(p.lock)

	p.goPurge()
	p.goTicktock()

	return p, nil
}

// purgeStaleWorkers 定期清理过期 worker（独立 goroutine）
func (p *poolCommon) purgeStaleWorkers() {
	ticker := time.NewTicker(p.options.ExpiryDuration)

	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&p.purgeDone, 1)
	}()

	purgeCtx := p.purgeCtx
	for {
		select {
		case <-purgeCtx.Done():
			return
		case <-ticker.C:
		}

		if p.IsClosed() {
			break
		}

		var isDormant bool
		p.lock.Lock()
		staleWorkers := p.workers.refresh(p.options.ExpiryDuration)
		n := p.Running()
		isDormant = n == 0 || n == len(staleWorkers)
		p.lock.Unlock()

		// 清理过期 worker
		for i := range staleWorkers {
			staleWorkers[i].finish()
			staleWorkers[i] = nil
		}

		// 如果所有 worker 都被清理，需要唤醒等待者
		if isDormant && p.Waiting() > 0 {
			p.cond.Broadcast()
		}
	}
}

const nowTimeUpdateInterval = 500 * time.Millisecond

// ticktock 定期更新当前时间（避免频繁 syscall）
func (p *poolCommon) ticktock() {
	ticker := time.NewTicker(nowTimeUpdateInterval)
	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&p.ticktockDone, 1)
	}()

	ticktockCtx := p.ticktockCtx
	for {
		select {
		case <-ticktockCtx.Done():
			return
		case <-ticker.C:
		}

		if p.IsClosed() {
			break
		}

		atomic.StoreInt64(&p.now, time.Now().UnixNano())
	}
}

func (p *poolCommon) goPurge() {
	if p.options.DisablePurge {
		return
	}
	p.purgeCtx, p.stopPurge = context.WithCancel(context.Background())
	go p.purgeStaleWorkers()
}

func (p *poolCommon) goTicktock() {
	atomic.StoreInt64(&p.now, time.Now().UnixNano())
	p.ticktockCtx, p.stopTicktock = context.WithCancel(context.Background())
	go p.ticktock()
}

func (p *poolCommon) nowTime() int64 {
	return atomic.LoadInt64(&p.now)
}

// Running 返回当前运行 worker 数
func (p *poolCommon) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free 返回可用 worker 数
func (p *poolCommon) Free() int {
	c := p.Cap()
	if c < 0 {
		return -1
	}
	return c - p.Running()
}

// Waiting 返回等待任务数量
func (p *poolCommon) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}

// Cap 返回池容量
func (p *poolCommon) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// Tune 动态调整容量（无限池和预分配池无效）
func (p *poolCommon) Tune(size int) {
	capacity := p.Cap()
	if capacity == -1 || size <= 0 || size == capacity || p.options.PreAlloc {
		return
	}
	atomic.StoreInt32(&p.capacity, int32(size))
	if size > capacity {
		if size-capacity == 1 {
			p.cond.Signal()
			return
		}
		p.cond.Broadcast()
	}
}

// IsClosed 判断池是否关闭
func (p *poolCommon) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

// Release 关闭池
func (p *poolCommon) Release() {
	if !atomic.CompareAndSwapInt32(&p.state, OPENED, CLOSED) {
		return
	}

	if p.stopPurge != nil {
		p.stopPurge()
		p.stopPurge = nil
	}
	if p.stopTicktock != nil {
		p.stopTicktock()
		p.stopTicktock = nil
	}

	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()

	p.cond.Broadcast()
}

// ReleaseTimeout 带超时关闭
func (p *poolCommon) ReleaseTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := p.ReleaseContext(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	return err
}

// ReleaseContext 带 context 关闭
func (p *poolCommon) ReleaseContext(ctx context.Context) error {
	if p.IsClosed() || (!p.options.DisablePurge && p.stopPurge == nil) || p.stopTicktock == nil {
		return ErrPoolClosed
	}

	p.Release()

	if ctx == nil {
		return nil
	}

	var purgeCh <-chan struct{}
	if !p.options.DisablePurge {
		purgeCh = p.purgeCtx.Done()
	} else {
		purgeCh = p.allDone
	}

	if p.Running() == 0 {
		p.once.Do(func() {
			close(p.allDone)
		})
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.allDone:
			<-purgeCh
			<-p.ticktockCtx.Done()
			if p.Running() == 0 &&
				(p.options.DisablePurge || atomic.LoadInt32(&p.purgeDone) == 1) &&
				atomic.LoadInt32(&p.ticktockDone) == 1 {
				return nil
			}
		}
	}
}

// Reboot 重启已关闭的池
func (p *poolCommon) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		atomic.StoreInt32(&p.purgeDone, 0)
		p.goPurge()
		atomic.StoreInt32(&p.ticktockDone, 0)
		p.goTicktock()
		p.allDone = make(chan struct{})
		p.once = &sync.Once{}
	}
}

func (p *poolCommon) addRunning(delta int) int {
	return int(atomic.AddInt32(&p.running, int32(delta)))
}

func (p *poolCommon) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}

// retrieveWorker 获取一个可用 worker
func (p *poolCommon) retrieveWorker() (w worker, err error) {
	p.lock.Lock()

retry:
	// 优先从队列获取
	if w = p.workers.detach(); w != nil {
		p.lock.Unlock()
		return
	}

	// 不够则创建
	if capacity := p.Cap(); capacity == -1 || capacity > p.Running() {
		w = p.workerCache.Get().(worker)
		w.run()
		p.lock.Unlock()
		return
	}

	// 非阻塞或达到上限
	if p.options.Nonblocking || (p.options.MaxBlockingTasks != 0 && p.Waiting() >= p.options.MaxBlockingTasks) {
		p.lock.Unlock()
		return nil, ErrPoolOverload
	}

	// 阻塞等待
	p.addWaiting(1)
	p.cond.Wait()
	p.addWaiting(-1)

	if p.IsClosed() {
		p.lock.Unlock()
		return nil, ErrPoolClosed
	}

	goto retry
}

// revertWorker 回收 worker（复用）
func (p *poolCommon) revertWorker(worker worker) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || p.IsClosed() {
		p.cond.Broadcast()
		return false
	}

	worker.setLastUsedTime(p.nowTime())

	p.lock.Lock()
	if p.IsClosed() {
		p.lock.Unlock()
		return false
	}
	if err := p.workers.insert(worker); err != nil {
		p.lock.Unlock()
		return false
	}
	p.cond.Signal()
	p.lock.Unlock()

	return true
}
