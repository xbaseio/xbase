package xqueue_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/xbaseio/xbase/utils/xqueue"
)

func TestLockFreeQueue(t *testing.T) {
	const taskNum = 10000
	q := xqueue.NewLockFreeQueue()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		for range taskNum {
			task := &xqueue.Task{}
			q.Enqueue(task)
		}
		wg.Done()
	}()
	go func() {
		for range taskNum {
			task := &xqueue.Task{}
			q.Enqueue(task)
		}
		wg.Done()
	}()

	var counter atomic.Int32
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				counter.Add(1)
			}
			if task == nil && counter.Load() == 2*taskNum {
				break
			}
		}
		wg.Done()
	}()
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				counter.Add(1)
			}
			if task == nil && counter.Load() == 2*taskNum {
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()

	t.Logf("sent and received all %d tasks", 2*taskNum)
}
