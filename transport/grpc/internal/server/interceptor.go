package server

import (
	"context"
	"runtime"

	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func recoverInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	defer func() {
		if err := recover(); err != nil {
			switch err.(type) {
			case runtime.Error:
				xlog.Logger().Panic("runtime panic", zap.Any("panic", err))
			default:
				xlog.Logger().Panic("panic error", zap.Any("panic", err))
			}
		}
	}()

	return handler(ctx, req)
}
