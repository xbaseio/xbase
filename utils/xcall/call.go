package xcall

import (
	"context"
	"runtime"
	"time"

	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

// Call 安全地调用函数
func Call(fn func()) {
	if fn == nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			switch err.(type) {
			case runtime.Error:
				xlog.Logger().Panic("runtime panic", zap.Any("panic", err))
			default:
				xlog.Sugar().Panicf("panic error: %v", err)
			}
		}
	}()

	fn()
}

// Go 执行单个协程
func Go(fn func()) {
	go Call(fn)
}

// GoWithTimeout 执行多个协程（附带超时时间）
func GoWithTimeout(timeout time.Duration, fns ...func()) {
	NewGoroutines().Add(fns...).Run(context.Background(), timeout)
}

// GoWithDeadline 执行多个协程（附带最后期限）
func GoWithDeadline(deadline time.Time, fns ...func()) {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	NewGoroutines().Add(fns...).Run(ctx)
}
