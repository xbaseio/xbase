package tcp_test

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/network/tcp"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xrand"
)

var pprofOnce sync.Once

func TestClient_Simple(t *testing.T) {
	client := tcp.NewClient()

	client.OnConnect(func(conn network.Conn) {
		log.Info("connection is opened")
	})

	client.OnDisconnect(func(conn network.Conn) {
		log.Info("connection is closed")
	})

	client.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			log.Errorf("unpack message failed: %v", err)
			return
		}

		log.Infof(
			"receive msg from server, cid: %d, seq: %d, game id: %d, msg: %s",
			conn.ID(),
			message.Seq,
			message.GameID,
			string(message.Buffer),
		)
	})

	conn, err := client.Dial()
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}
	defer conn.Close()

	msg, err := packet.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello server~~"),
	})
	if err != nil {
		t.Fatalf("pack message failed: %v", err)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range 200 {
		<-ticker.C

		if err = conn.Push(msg); err != nil {
			log.Errorf("push message failed: %v", err)
			return
		}
	}
}

func TestClient_Benchmark(t *testing.T) {
	samples := []struct {
		c    int // 并发连接数
		n    int // 请求数
		size int // 数据包大小
	}{
		{c: 50, n: 1000000, size: 1024},
		{c: 100, n: 1000000, size: 1024},
		{c: 200, n: 1000000, size: 1024},
		{c: 300, n: 1000000, size: 1024},
		{c: 400, n: 1000000, size: 1024},
		{c: 500, n: 1000000, size: 1024},
		{c: 1000, n: 1000000, size: 2 * 1024},
	}

	startPprof()

	for _, sample := range samples {
		doPressureTest(sample.c, sample.n, sample.size)
	}
}

func startPprof() {
	pprofOnce.Do(func() {
		go func() {
			if err := http.ListenAndServe(":8090", nil); err != nil {
				log.Errorf("pprof server start failed: %v", err)
			}
		}()
	})
}

// 执行压力测试
func doPressureTest(concurrency int, requests int, size int) {
	var (
		totalSent int64
		totalRecv int64
		totalFail int64
	)

	client := tcp.NewClient()

	client.OnReceive(func(conn network.Conn, data []byte) {
		// 防止异常情况下收到超过 requests 的响应导致统计失真
		for {
			old := atomic.LoadInt64(&totalRecv)
			if old >= int64(requests) {
				return
			}

			if atomic.CompareAndSwapInt64(&totalRecv, old, old+1) {
				return
			}
		}
	})

	payload := []byte(xrand.Letters(size))

	msg, err := packet.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    payload,
	})
	if err != nil {
		log.Errorf("pack message failed: %v", err)
		return
	}

	conns := dialClients(client, concurrency)
	if len(conns) == 0 {
		log.Errorf("no tcp connection available")
		return
	}

	actualConcurrency := len(conns)

	jobs := make(chan struct{}, minInt(8192, actualConcurrency*8))

	var workerWG sync.WaitGroup
	workerWG.Add(actualConcurrency)

	for _, conn := range conns {
		go func(conn network.Conn) {
			defer workerWG.Done()

			for range jobs {
				if err := conn.Push(msg); err != nil {
					fail := atomic.AddInt64(&totalFail, 1)
					if fail <= 10 {
						log.Errorf("push message failed: %v", err)
					}
					continue
				}

				atomic.AddInt64(&totalSent, 1)
			}
		}(conn)
	}

	startTime := time.Now()

	for range requests {
		jobs <- struct{}{}
	}

	close(jobs)

	workerWG.Wait()

	sent := atomic.LoadInt64(&totalSent)

	ok := waitRecv(&totalRecv, sent, 5*time.Minute)
	if !ok {
		log.Warnf(
			"wait receive timeout, sent: %d, recv: %d, fail: %d",
			sent,
			atomic.LoadInt64(&totalRecv),
			atomic.LoadInt64(&totalFail),
		)
	}

	totalTime := time.Since(startTime).Seconds()

	for _, conn := range conns {
		_ = conn.Close(true)
	}

	recv := atomic.LoadInt64(&totalRecv)
	fail := atomic.LoadInt64(&totalFail)

	fmt.Printf("server               : %s\n", client.Protocol())
	fmt.Printf("concurrency          : %d\n", actualConcurrency)
	fmt.Printf("latency              : %.6fs\n", totalTime)
	fmt.Printf("data size            : %s\n", convBytes(int64(size)))
	fmt.Printf("sent requests        : %d\n", sent)
	fmt.Printf("received requests    : %d\n", recv)
	fmt.Printf("failed requests      : %d\n", fail)
	fmt.Printf("lost responses       : %d\n", sent-recv)
	fmt.Printf("throughput (TPS)     : %d\n", int64(float64(recv)/totalTime))
	fmt.Printf("--------------------------------\n")
}

func dialClients(client network.Client, concurrency int) []network.Conn {
	conns := make([]network.Conn, 0, concurrency)

	maxAttempts := max(concurrency*10, 10)

	for attempts := 0; len(conns) < concurrency && attempts < maxAttempts; attempts++ {
		conn, err := client.Dial()
		if err != nil {
			log.Errorf("client dial failed: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		conns = append(conns, conn)
	}

	if len(conns) < concurrency {
		log.Warnf("dial connections not enough, want: %d, got: %d", concurrency, len(conns))
	}

	return conns
}

func waitRecv(totalRecv *int64, expect int64, timeout time.Duration) bool {
	if expect <= 0 {
		return true
	}

	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)

	defer timer.Stop()
	defer ticker.Stop()

	for {
		if atomic.LoadInt64(totalRecv) >= expect {
			return true
		}

		select {
		case <-ticker.C:
		case <-timer.C:
			return atomic.LoadInt64(totalRecv) >= expect
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func convBytes(bytes int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)

	switch {
	case bytes < KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/KB)
	case bytes < GB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/GB)
	case bytes < TB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/GB)
	default:
		return fmt.Sprintf("%.2fTB", float64(bytes)/TB)
	}
}
