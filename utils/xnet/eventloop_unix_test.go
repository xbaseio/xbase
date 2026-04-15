//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	"context"
	"net"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/utils/xpool/xgoroutine"
	"github.com/stretchr/testify/assert"
)

//
// =========================
// LoadBalancer 注册扩展
// =========================
//

// register 时注入初始化连接（用于压测）
func (lb *roundRobinLoadBalancer) register(el *eventloop) {
	lb.baseLoadBalancer.register(el)
	registerInitConn(el)
}

func (lb *leastConnectionsLoadBalancer) register(el *eventloop) {
	lb.baseLoadBalancer.register(el)
	registerInitConn(el)
}

func (lb *sourceAddrHashLoadBalancer) register(el *eventloop) {
	lb.baseLoadBalancer.register(el)
	registerInitConn(el)
}

// registerInitConn 为每个 eventloop 注入假连接（用于模拟大规模连接）
func registerInitConn(el *eventloop) {
	count := int(atomic.LoadInt32(&nowEventLoopInitConn))

	for i := 0; i < count; i++ {
		c := newStreamConn(
			"tcp",
			i,
			el,
			&unix.SockaddrInet4{},
			&net.TCPAddr{},
			&net.TCPAddr{},
		)
		el.connections.addConn(c, el.idx)
	}
}

//
// =========================
// 全局测试变量
// =========================
//

// 初始化 fake conn 数量（测试用，结束后必须归零）
var nowEventLoopInitConn int32

// 是否启用大规模 GC 测试（手动打开）
var testBigGC = false

//
// =========================
// Benchmark（GC 测试）
// =========================
//

func BenchmarkGC4El100k(b *testing.B) {
	runBenchmarkGC(b, 4, 100000)
}

func BenchmarkGC4El200k(b *testing.B) {
	runBenchmarkGC(b, 4, 200000)
}

func BenchmarkGC4El500k(b *testing.B) {
	runBenchmarkGC(b, 4, 500000)
}

// runBenchmarkGC 统一 benchmark 入口
func runBenchmarkGC(b *testing.B, elNum int, connNum int32) {
	oldGC := debug.SetGCPercent(-1)

	ts := benchServeGC(b, "tcp", ":0", true, elNum, connNum)

	b.Run("Run-GC", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runtime.GC()
		}
	})

	_ = ts.eng.Stop(context.Background())
	debug.SetGCPercent(oldGC)
}

//
// =========================
// Benchmark Server
// =========================
//

func benchServeGC(
	b *testing.B,
	network, addr string,
	async bool,
	elNum int,
	initConnCount int32,
) *benchmarkServerGC {

	ts := &benchmarkServerGC{
		tester:        b,
		network:       network,
		addr:          addr,
		async:         async,
		elNum:         elNum,
		initConnCount: initConnCount,
		initOk:        make(chan struct{}),
	}

	nowEventLoopInitConn = initConnCount

	_ = xgoroutine.DefaultWorkerPool.Submit(func() {
		err := Run(
			ts,
			network+"://"+addr,
			WithLockOSThread(async),
			WithNumEventLoop(elNum),
			WithTCPKeepAlive(time.Minute),
			WithTCPNoDelay(TCPDelay),
		)
		assert.NoError(b, err)
		nowEventLoopInitConn = 0
	})

	<-ts.initOk
	return ts
}

// benchmarkServerGC 用于 GC benchmark
type benchmarkServerGC struct {
	*BuiltinEventEngine

	tester        *testing.B
	eng           Engine
	network       string
	addr          string
	async         bool
	elNum         int
	initConnCount int32
	initOk        chan struct{}
}

// OnBoot 等待所有 fake conn 初始化完成
func (s *benchmarkServerGC) OnBoot(eng Engine) (action Action) {
	s.eng = eng

	_ = xgoroutine.DefaultWorkerPool.Submit(func() {
		for s.eng.eng.eventLoops.len() != s.elNum ||
			s.eng.CountConnections() != s.elNum*int(s.initConnCount) {
			time.Sleep(time.Millisecond)
		}
		close(s.initOk)
	})

	return
}

//
// =========================
// Test（GC 行为测试）
// =========================
//

// TestServeGC 用于手动 GC 压测（默认关闭大规模测试）
func TestServeGC(t *testing.T) {
	t.Run("gc-loop", func(t *testing.T) {

		runTestCase := func(name string, elNum int, connNum int32) {
			t.Run(name, func(t *testing.T) {
				if connNum >= 100000 && !testBigGC {
					t.Skipf("Skip big GC test, testBigGC=%t", testBigGC)
				}
				testServeGC(t, "tcp", ":0", true, true, elNum, connNum)
			})
		}

		runTestCase("1-loop-10000", 1, 10000)
		runTestCase("1-loop-100000", 1, 100000)
		runTestCase("1-loop-1000000", 1, 1000000)

		runTestCase("2-loop-10000", 2, 10000)
		runTestCase("2-loop-100000", 2, 100000)
		runTestCase("2-loop-1000000", 2, 1000000)

		runTestCase("4-loop-10000", 4, 10000)
		runTestCase("4-loop-100000", 4, 100000)
		runTestCase("4-loop-1000000", 4, 1000000)

		runTestCase("16-loop-10000", 16, 10000)
		runTestCase("16-loop-100000", 16, 100000)
		runTestCase("16-loop-1000000", 16, 1000000)
	})
}

//
// =========================
// Test Server
// =========================
//

func testServeGC(
	t *testing.T,
	network, addr string,
	multicore, async bool,
	elNum int,
	initConnCount int32,
) {

	ts := &testServerGC{
		tester:    t,
		network:   network,
		addr:      addr,
		multicore: multicore,
		async:     async,
		elNum:     elNum,
	}

	nowEventLoopInitConn = initConnCount

	err := Run(
		ts,
		network+"://"+addr,
		WithLockOSThread(async),
		WithMulticore(multicore),
		WithNumEventLoop(elNum),
		WithTCPKeepAlive(time.Minute),
		WithTCPNoDelay(TCPDelay),
	)
	assert.NoError(t, err)

	nowEventLoopInitConn = 0
}

// testServerGC 用于测试 GC 行为
type testServerGC struct {
	*BuiltinEventEngine

	tester    *testing.T
	eng       Engine
	network   string
	addr      string
	multicore bool
	async     bool
	elNum     int
}

// OnBoot 启动 GC 压测协程
func (s *testServerGC) OnBoot(eng Engine) (action Action) {
	s.eng = eng

	gcSeconds := 5
	if testBigGC {
		gcSeconds = 10
	}

	err := xgoroutine.DefaultWorkerPool.Submit(func() {
		s.runGC(gcSeconds)
	})
	assert.NoError(s.tester, err)

	return
}

// runGC 每秒执行一次 GC 并统计耗时
func (s *testServerGC) runGC(seconds int) {
	defer func() {
		_ = s.eng.Stop(context.Background())
		runtime.GC()
	}()

	var totalGC time.Duration
	var count time.Duration

	start := time.Now()

	for range time.Tick(time.Second) {
		count++

		now := time.Now()
		runtime.GC()
		gcTime := time.Since(now)

		totalGC += gcTime

		s.tester.Log(
			s.tester.Name(),
			s.network,
			"server gc:",
			gcTime,
			"average gc time:",
			totalGC/count,
		)

		if time.Since(start) >= time.Second*time.Duration(seconds) {
			break
		}
	}
}
