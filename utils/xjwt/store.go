package xjwt

import (
	"context"
	"time"
)

type (
	Store interface {
		Get(ctx context.Context, key interface{}) (interface{}, error)

		Set(ctx context.Context, key interface{}, value interface{}, duration time.Duration) error

		Remove(ctx context.Context, keys ...interface{}) (value interface{}, err error)
	}
)
