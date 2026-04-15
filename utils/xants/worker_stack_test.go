package xants

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewWorkerStack 测试 worker 栈初始化后的基础状态。
func TestNewWorkerStack(t *testing.T) {
	size := 100
	q := newWorkerStack(size)
	require.EqualValues(t, 0, q.len(), "Len error")
	require.Equal(t, true, q.isEmpty(), "IsEmpty error")
	require.Nil(t, q.detach(), "Dequeue error")
}

// TestWorkerStack 测试 worker 栈的入栈、过期清理等基础行为。
func TestWorkerStack(t *testing.T) {
	q := newWorkerQueue(queueType(-1), 0)

	for i := 0; i < 5; i++ {
		err := q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
		if err != nil {
			break
		}
	}
	require.EqualValues(t, 5, q.len(), "Len error")

	expired := time.Now().UnixNano()

	err := q.insert(&goWorker{lastUsed: expired})
	if err != nil {
		t.Fatal("Enqueue error")
	}

	time.Sleep(time.Second)

	for i := 0; i < 6; i++ {
		err := q.insert(&goWorker{lastUsed: time.Now().UnixNano()})
		if err != nil {
			t.Fatal("Enqueue error")
		}
	}
	require.EqualValues(t, 12, q.len(), "Len error")
	q.refresh(time.Second)
	require.EqualValues(t, 6, q.len(), "Len error")
}

// TestSearch 测试二分查找逻辑。
//
// 看起来 Windows 上的 time.Now() 存在一些异常表现，
// 目前不确定是否是 Windows 平台本身的问题，
// 因此建议将该测试单独拆到非 Windows 文件中。
func TestSearch(t *testing.T) {
	q := newWorkerStack(0)

	currTime := time.Now().UnixNano()

	// 1
	expiry1 := currTime
	currTime++
	_ = q.insert(&goWorker{lastUsed: currTime})

	require.EqualValues(t, 0, q.binarySearch(0, q.len()-1, currTime), "index should be 0")
	require.EqualValues(t, -1, q.binarySearch(0, q.len()-1, expiry1), "index should be -1")

	// 2
	currTime++
	expiry2 := currTime
	currTime++
	_ = q.insert(&goWorker{lastUsed: currTime})

	require.EqualValues(t, -1, q.binarySearch(0, q.len()-1, expiry1), "index should be -1")
	require.EqualValues(t, 0, q.binarySearch(0, q.len()-1, expiry2), "index should be 0")
	require.EqualValues(t, 1, q.binarySearch(0, q.len()-1, currTime), "index should be 1")

	// more
	for i := 0; i < 5; i++ {
		currTime++
		_ = q.insert(&goWorker{lastUsed: currTime})
	}

	currTime++
	expiry3 := currTime

	_ = q.insert(&goWorker{lastUsed: expiry3})

	for i := 0; i < 10; i++ {
		currTime++
		_ = q.insert(&goWorker{lastUsed: currTime})
	}

	require.EqualValues(t, 7, q.binarySearch(0, q.len()-1, expiry3), "index should be 7")
}
