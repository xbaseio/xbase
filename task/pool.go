package task

import (
	"github.com/xbaseio/xbase/utils/xants"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type Pool interface {
	// AddTask 添加任务
	AddTask(task func()) error
	// Release 释放任务
	Release()
}

var globalPool Pool

func init() {
	SetPool(NewPool())
}

type defaultPool struct {
	pool *xants.Pool
}

func NewPool(opts ...Option) *defaultPool {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	p := &defaultPool{}
	p.pool, _ = xants.NewPool(o.size,
		xants.WithLogger(&logger{}),
		xants.WithNonblocking(o.nonblocking),
		xants.WithDisablePurge(o.disablePurge),
	)

	return p
}

// AddTask 添加任务
func (p *defaultPool) AddTask(task func()) error {
	return p.pool.Submit(task)
}

// Release 释放任务
func (p *defaultPool) Release() {
	p.pool.Release()
}

// SetPool 设置任务池
func SetPool(pool Pool) {
	if globalPool != nil {
		globalPool.Release()
	}
	globalPool = pool
}

// GetPool 获取任务池
func GetPool() Pool {
	return globalPool
}

// AddTask 添加任务
func AddTask(task func()) {
	if globalPool == nil {
		xcall.Go(task)
		return
	}

	if err := globalPool.AddTask(task); err != nil {
		xcall.Go(task)
		xlog.Logger().Warn("add task to the task pool failed", zap.Error(err))
		return
	}
}

// Release 释放任务
func Release() {
	if globalPool != nil {
		globalPool.Release()
	}
}

type logger struct {
}

func (l *logger) Printf(format string, args ...any) {
	xlog.Logger().Info(format, zap.Any("args", []any{args}))
}
