//go:build !race

package xants

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewLoopQueue 测试队列初始化及基础状态
func TestNewLoopQueue(t *testing.T) {
	size := 100
	q := newWorkerLoopQueue(size)

	require.EqualValues(t, 0, q.len(), "Len error")
	require.Equal(t, true, q.isEmpty(), "IsEmpty error")
	require.Nil(t, q.detach(), "Dequeue error")

	// size=0 应返回 nil
	require.Nil(t, newWorkerLoopQueue(0))
}

// TestLoopQueue 测试循环队列基本操作（入队/出队/容量/过期清理）
func TestLoopQueue(t *testing.T) {
	size := 10
	q := newWorkerLoopQueue(size)

	// 插入 5 个
	for i := 0; i < 5; i++ {
		err := q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
		if err != nil {
			break
		}
	}
	require.EqualValues(t, 5, q.len(), "Len error")

	// 出队 1 个
	_ = q.detach()
	require.EqualValues(t, 4, q.len(), "Len error")

	time.Sleep(time.Second)

	// 再插入直到接近满
	for i := 0; i < 6; i++ {
		err := q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
		if err != nil {
			break
		}
	}
	require.EqualValues(t, 10, q.len(), "Len error")

	// 超容量插入应报错
	err := q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
	require.Error(t, err, "Enqueue, error")

	// 清理过期 worker
	q.refresh(time.Second)
	require.EqualValuesf(t, 6, q.len(), "Len error: %d", q.len())
}

// TestRotatedQueueSearch 测试环形队列中的二分查找（包含旋转场景）
func TestRotatedQueueSearch(t *testing.T) {
	size := 10
	q := newWorkerLoopQueue(size)

	currTime := time.Now().UnixNano()

	// --- 基础测试 ---
	expiry1 := currTime
	currTime++
	_ = q.insert(&goWorker{lastUsed: currTime})

	require.EqualValues(t, 0, q.binarySearch(currTime), "index should be 0")
	require.EqualValues(t, -1, q.binarySearch(expiry1), "index should be -1")

	// --- 增加一个元素 ---
	currTime++
	expiry2 := currTime
	currTime++
	_ = q.insert(&goWorker{lastUsed: currTime})

	require.EqualValues(t, -1, q.binarySearch(expiry1), "index should be -1")
	require.EqualValues(t, 0, q.binarySearch(expiry2), "index should be 0")
	require.EqualValues(t, 1, q.binarySearch(currTime), "index should be 1")

	// --- 填充更多元素 ---
	for i := 0; i < 5; i++ {
		currTime++
		_ = q.insert(&goWorker{lastUsed: currTime})
	}

	currTime++
	expiry3 := currTime
	_ = q.insert(&goWorker{lastUsed: expiry3})

	var err error
	for err != errQueueIsFull {
		currTime++
		err = q.insert(&goWorker{lastUsed: currTime})
	}

	require.EqualValues(t, 7, q.binarySearch(expiry3), "index should be 7")

	// --- 队列旋转（head 移动）---
	for i := 0; i < 6; i++ {
		_ = q.detach()
	}

	currTime++
	expiry4 := currTime
	_ = q.insert(&goWorker{lastUsed: expiry4})

	for i := 0; i < 4; i++ {
		currTime++
		_ = q.insert(&goWorker{lastUsed: currTime})
	}

	// head=6, tail=5
	require.EqualValues(t, 0, q.binarySearch(expiry4), "index should be 0")

	for i := 0; i < 3; i++ {
		_ = q.detach()
	}

	currTime++
	expiry5 := currTime
	_ = q.insert(&goWorker{lastUsed: expiry5})

	// head=6, tail=5
	require.EqualValues(t, 5, q.binarySearch(expiry5), "index should be 5")

	for i := 0; i < 3; i++ {
		currTime++
		_ = q.insert(&goWorker{lastUsed: currTime})
	}

	// head=9, tail=9
	require.EqualValues(t, -1, q.binarySearch(expiry2), "index should be -1")
	require.EqualValues(t, 9, q.binarySearch(q.items[9].lastUsedTime()), "index should be 9")
	require.EqualValues(t, 8, q.binarySearch(currTime), "index should be 8")
}

// TestRetrieveExpiry 测试过期 worker 的批量回收逻辑
func TestRetrieveExpiry(t *testing.T) {
	size := 10
	q := newWorkerLoopQueue(size)
	expirew := make([]worker, 0)

	u, _ := time.ParseDuration("1s")

	// 场景1：前半部分过期
	for i := 0; i < size/2; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
	}
	expirew = append(expirew, q.items[:size/2]...)

	time.Sleep(u)

	for i := 0; i < size/2; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
	}

	workers := q.refresh(u)
	require.EqualValues(t, expirew, workers, "expired workers aren't right")

	// 场景2：后半部分过期
	time.Sleep(u)

	for i := 0; i < size/2; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
	}

	expirew = expirew[:0]
	expirew = append(expirew, q.items[size/2:]...)

	workers2 := q.refresh(u)
	require.EqualValues(t, expirew, workers2, "expired workers aren't right")

	// 场景3：部分为空 + 旋转
	for i := 0; i < size/2; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
	}
	for i := 0; i < size/2; i++ {
		_ = q.detach()
	}
	for i := 0; i < 3; i++ {
		_ = q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
	}

	time.Sleep(u)

	expirew = expirew[:0]
	expirew = append(expirew, q.items[0:3]...)
	expirew = append(expirew, q.items[size/2:]...)

	workers3 := q.refresh(u)
	require.EqualValues(t, expirew, workers3, "expired workers aren't right")
}
