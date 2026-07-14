package xjwt

import (
	"context"
	"time"
)

type (
	Store interface {
		Get(ctx context.Context, key any) (any, error)

		Set(ctx context.Context, key any, value any, duration time.Duration) error

		Remove(ctx context.Context, keys ...any) (value any, err error)
	}
)
