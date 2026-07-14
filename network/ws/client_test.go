package ws_test

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
	"github.com/xbaseio/xbase/network/ws"
	"github.com/xbaseio/xbase/packet"
)

var pprofOnce sync.Once

func TestClient_Dial(t *testing.T) {
	client := ws.NewClient()

	client.OnConnect(func(conn network.Conn) {
		t.Log("connection is opened")
	})

	client.OnDisconnect(func(conn network.Conn) {
		t.Log("connection is closed")
	})

	client.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			t.Error(err)
			return
		}

		t.Logf(
			"receive msg from server, connection id: %d, seq: %d, game id: %d, msg: %s",
			conn.ID(),
			message.Seq,
			message.GameID,
			string(message.Buffer),
		)
	})

	conn, err := client.Dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close(true)

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

	for range 5 {
		<-ticker.C

		if err = conn.Push(msg); err != nil {
			t.Errorf("push message failed: %v", err)
			return
		}
	}
}

func TestNewClient(t *testing.T) {
	client := ws.NewClient()

	client.OnConnect(func(conn network.Conn) {
		log.Info("connection is opened")
	})

	client.OnDisconnect(func(conn network.Conn) {
		log.Info("connection is closed")
	})

	client.OnReceive(func(conn network.Conn, data []byte) {
		message, _, err := packet.UnpackMessage(data)
		if err != nil {
			t.Error(err)
			return
		}

		t.Logf(
			"receive msg from server, connection id: %d, seq: %d, game id: %d, msg: %s",
			conn.ID(),
			message.Seq,
			message.GameID,
			string(message.Buffer),
		)
	})

	conn, err := client.Dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close(true)

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

	for range 5 {
		<-ticker.C

		if err = conn.Push(msg); err != nil {
			t.Errorf("push message failed: %v", err)
			return
		}
	}
}

func TestClient_Benchmark(t *testing.T) {
	startPprof()

	samples := []struct {
		concurrency int
		total       int
	}{
		{concurrency: 100, total: 1000000},
		{concurrency: 300, total: 1000000},
		{concurrency: 500, total: 1000000},
		{concurrency: 1000, total: 1000000},
	}

	for _, sample := range samples {
		doPressureTest(t, sample.concurrency, sample.total)
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
func doPressureTest(t *testing.T, concurrency int, total int) {
	var (
		totalSent int64
		totalRecv int64
		totalFail int64
	)

	client := ws.NewClient()

	client.OnReceive(func(conn network.Conn, data []byte) {
		for {
			old := atomic.LoadInt64(&totalRecv)
			if old >= int64(total) {
				return
			}

			if atomic.CompareAndSwapInt64(&totalRecv, old, old+1) {
				return
			}
		}
	})

	msg, err := packet.PackMessage(&packet.Message{
		Seq:       1,
		GameID:    1,
		MessageID: 1001,
		Buffer:    []byte("hello server~~"),
	})
	if err != nil {
		t.Fatalf("pack message failed: %v", err)
	}

	conns := dialClients(func() (network.Conn, error) {
		return client.Dial()
	}, concurrency)

	if len(conns) == 0 {
		t.Fatalf("no websocket connection available")
	}

	actualConcurrency := len(conns)

	// 不要用 total 作为 channel 容量，100w 容量没必要。
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

	for range total {
		jobs <- struct{}{}
	}

	close(jobs)

	workerWG.Wait()

	sent := atomic.LoadInt64(&totalSent)

	if ok := waitRecv(&totalRecv, sent, 5*time.Minute); !ok {
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

	fmt.Printf("server               : %s\n", "websocket")
	fmt.Printf("concurrency          : %d\n", actualConcurrency)
	fmt.Printf("latency              : %.6fs\n", totalTime)
	fmt.Printf("sent requests        : %d\n", sent)
	fmt.Printf("received requests    : %d\n", recv)
	fmt.Printf("failed requests      : %d\n", fail)
	fmt.Printf("lost responses       : %d\n", sent-recv)
	fmt.Printf("throughput (TPS)     : %d\n", int64(float64(recv)/totalTime))
	fmt.Printf("--------------------------------\n")
}

func dialClients(dial func() (network.Conn, error), concurrency int) []network.Conn {
	conns := make([]network.Conn, 0, concurrency)

	maxAttempts := max(concurrency*10, 10)

	for attempts := 0; len(conns) < concurrency && attempts < maxAttempts; attempts++ {
		conn, err := dial()
		if err != nil {
			log.Errorf("client dial failed: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		conns = append(conns, conn)

		// 防止瞬间建太多连接，WebSocket 握手压力过大。
		if len(conns)%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
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
