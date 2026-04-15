package xants

import (
	"runtime/debug"
)

// goWorkerWithFuncGeneric 是真正执行任务的 worker。
// 它会启动一个 goroutine，接收任务参数并调用统一函数执行。
type goWorkerWithFuncGeneric[T any] struct {
	worker

	// pool 表示拥有该 worker 的协程池。
	pool *PoolWithFuncGeneric[T]

	// arg 表示待处理任务参数。
	arg chan T

	// exit 用于通知 goroutine 退出。
	exit chan struct{}

	// lastUsed 表示该 worker 最近一次被使用的时间。
	lastUsed int64
}

// run 启动 worker goroutine，循环执行任务处理流程。
func (w *goWorkerWithFuncGeneric[T]) run() {
	w.pool.addRunning(1)

	go func() {
		defer func() {
			if w.pool.addRunning(-1) == 0 && w.pool.IsClosed() {
				w.pool.once.Do(func() {
					close(w.pool.allDone)
				})
			}

			w.pool.workerCache.Put(w)

			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from panic: %v\n%s\n", p, debug.Stack())
				}
			}

			// 在这里调用 Signal()，避免有协程一直等待可用 worker。
			w.pool.cond.Signal()
		}()

		for {
			select {
			case <-w.exit:
				return

			case arg := <-w.arg:
				w.pool.fn(arg)
				if ok := w.pool.revertWorker(w); !ok {
					return
				}
			}
		}
	}()
}

func (w *goWorkerWithFuncGeneric[T]) finish() {
	w.exit <- struct{}{}
}

func (w *goWorkerWithFuncGeneric[T]) lastUsedTime() int64 {
	return w.lastUsed
}

func (w *goWorkerWithFuncGeneric[T]) setLastUsedTime(t int64) {
	w.lastUsed = t
}
