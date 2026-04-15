package xcall

import (
	"context"
	"sync"
	"time"
)

// 并发执行多个任务，并等待所有任务完成或超时 并返回第一个错误（如果有）
type Task func(ctx context.Context) error

type Group struct {
	tasks []Task
}

func (g *Group) Add(tasks ...Task) *Group {
	g.tasks = append(g.tasks, tasks...)
	return g
}

func (g *Group) Run(ctx context.Context, timeout time.Duration) error {
	if len(g.tasks) == 0 {
		return nil
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(g.tasks))

	wg.Add(len(g.tasks))

	for _, t := range g.tasks {
		task := t
		go func() {
			defer wg.Done()

			if err := task(ctx); err != nil {
				errCh <- err
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		close(errCh)
		for err := range errCh {
			if err != nil {
				return err
			}
		}
		return nil
	}
}
