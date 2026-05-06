package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xbaseio/xbase/cache"
	"github.com/xbaseio/xbase/core/tls"
	"github.com/xbaseio/xbase/utils/xconv"
	"github.com/xbaseio/xbase/utils/xrand"
	"github.com/xbaseio/xbase/utils/xreflect"
	"github.com/xbaseio/xbase/xerrors"
	"golang.org/x/sync/singleflight"
)

type Cache struct {
	err     error
	opts    *options
	builtin bool
	sfg     singleflight.Group
}

func NewCache(opts ...Option) *Cache {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	c := &Cache{}

	defer func() {
		if c.err == nil {
			c.opts = o
		}
	}()

	if o.client == nil {
		options := &redis.UniversalOptions{
			Addrs:      o.addrs,
			DB:         o.db,
			Username:   o.username,
			Password:   o.password,
			MaxRetries: o.maxRetries,
		}

		if o.certFile != "" && o.keyFile != "" && o.caFile != "" {
			if options.TLSConfig, c.err = tls.MakeRedisTLSConfig(o.certFile, o.keyFile, o.caFile); c.err != nil {
				return c
			}
		}

		o.client, c.builtin = redis.NewUniversalClient(options), true
	}

	return c
}

// Has 检测缓存是否存在
func (c *Cache) Has(ctx context.Context, key string) (bool, error) {
	key = c.AddPrefix(key)

	val, err, _ := c.sfg.Do(key, func() (any, error) {
		return c.opts.client.Get(ctx, key).Result()
	})
	if err != nil {
		if xerrors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	if val.(string) == c.opts.nilValue {
		return false, nil
	}

	return true, nil
}

// Get 获取缓存值
func (c *Cache) Get(ctx context.Context, key string, def ...any) cache.Result {
	key = c.AddPrefix(key)

	val, err, _ := c.sfg.Do(key, func() (any, error) {
		return c.opts.client.Get(ctx, key).Result()
	})
	if err != nil && !xerrors.Is(err, redis.Nil) {
		return cache.NewResult(nil, err)
	}

	if xerrors.Is(err, redis.Nil) || val == c.opts.nilValue {
		if len(def) > 0 {
			return cache.NewResult(def[0])
		} else {
			return cache.NewResult(nil, xerrors.ErrNil)
		}
	}

	return cache.NewResult(val)
}

// Set 设置缓存值
func (c *Cache) Set(ctx context.Context, key string, value any, expiration ...time.Duration) error {
	if len(expiration) > 0 {
		return c.opts.client.Set(ctx, c.AddPrefix(key), xconv.String(value), expiration[0]).Err()
	} else {
		return c.opts.client.Set(ctx, c.AddPrefix(key), xconv.String(value), redis.KeepTTL).Err()
	}
}

func (c *Cache) GetSet(ctx context.Context, key string, fn cache.SetValueFunc, expiration ...time.Duration) cache.Result {
	key = c.AddPrefix(key)

	// 1. 先读 Redis，singleflight 防止同一个 key 并发打 Redis
	val, err, _ := c.sfg.Do(key, func() (any, error) {
		return c.opts.client.Get(ctx, key).Result()
	})

	if err != nil && !xerrors.Is(err, redis.Nil) {
		return cache.NewResult(nil, err)
	}

	if err == nil {
		if val == c.opts.nilValue {
			return cache.NewResult(nil, xerrors.ErrNil)
		}
		return cache.NewResult(val)
	}

	// 2. Redis miss 后，singleflight 防止并发回源
	rst, err, _ := c.sfg.Do(key+":set", func() (any, error) {
		val, err := fn()
		if err != nil {
			return cache.NewResult(nil, err), nil
		}

		// 空值缓存，还是走单独的 nilExpiration
		if val == nil || xreflect.IsNil(val) {
			err = c.opts.client.Set(ctx, key, c.opts.nilValue, c.opts.nilExpiration).Err()
			if err != nil {
				return cache.NewResult(nil, err), nil
			}

			return cache.NewResult(nil, xerrors.ErrNil), nil
		}

		ttl := c.GetExpiration(expiration...)

		err = c.opts.client.Set(ctx, key, xconv.String(val), ttl).Err()
		if err != nil {
			return cache.NewResult(nil, err), nil
		}

		return cache.NewResult(val, nil), nil
	})

	if err != nil {
		return cache.NewResult(nil, err)
	}

	return rst.(cache.Result)
}
func (c *Cache) GetExpiration(expiration ...time.Duration) time.Duration {
	// 显式传了，就使用传入的缓存时间。
	// 注意：传 0 表示 Redis 永不过期。
	if len(expiration) > 0 {
		return expiration[0]
	}

	minExpiration := c.opts.minExpiration
	maxExpiration := c.opts.maxExpiration

	// 都没配置，默认永不过期
	if minExpiration <= 0 && maxExpiration <= 0 {
		return 0
	}

	// 只配置了 max
	if minExpiration <= 0 {
		return maxExpiration
	}

	// min >= max 时，避免随机函数异常，直接用 min
	if maxExpiration <= minExpiration {
		return minExpiration
	}

	// 默认走随机过期时间，避免大量 key 同时过期造成缓存雪崩
	return time.Duration(xrand.Int64(int64(minExpiration), int64(maxExpiration)))
}

// Delete 删除缓存
func (c *Cache) Delete(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	for i, key := range keys {
		keys[i] = c.AddPrefix(key)
	}

	return c.opts.client.Del(ctx, keys...).Result()
}

// IncrInt 整数自增
func (c *Cache) IncrInt(ctx context.Context, key string, value int64) (int64, error) {
	return c.opts.client.IncrBy(ctx, c.AddPrefix(key), value).Result()
}

// IncrFloat 浮点数自增
func (c *Cache) IncrFloat(ctx context.Context, key string, value float64) (float64, error) {
	return c.opts.client.IncrByFloat(ctx, c.AddPrefix(key), value).Result()
}

// DecrInt 整数自减
func (c *Cache) DecrInt(ctx context.Context, key string, value int64) (int64, error) {
	return c.opts.client.DecrBy(ctx, c.AddPrefix(key), value).Result()
}

// DecrFloat 浮点数自减
func (c *Cache) DecrFloat(ctx context.Context, key string, value float64) (float64, error) {
	return c.opts.client.IncrByFloat(ctx, c.AddPrefix(key), -value).Result()
}

// AddPrefix 添加Key前缀
func (c *Cache) AddPrefix(key string) string {
	if c.opts.prefix == "" {
		return key
	} else {
		return c.opts.prefix + ":" + key
	}
}

// Client 获取客户端
func (c *Cache) Client() any {
	return c.opts.client
}

// Close 关闭缓存
func (c *Cache) Close() error {
	if c.builtin {
		return c.opts.client.Close()
	}

	return nil
}
