package memcache

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/xbaseio/xbase/cache"
	"github.com/xbaseio/xbase/utils/xconv"
	"github.com/xbaseio/xbase/utils/xrand"
	"github.com/xbaseio/xbase/utils/xreflect"
	"github.com/xbaseio/xbase/xerrors"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

type Cache struct {
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
	c.opts = o

	if o.client == nil {
		o.client, c.builtin = memcache.New(o.addrs...), true
	}

	return c
}

// Has 检测缓存是否存在
func (c *Cache) Has(ctx context.Context, key string) (bool, error) {
	key = c.AddPrefix(key)

	val, err, _ := c.sfg.Do(key, func() (any, error) {
		item, err := c.opts.client.Get(key)
		if err != nil {
			return nil, err
		}

		return xconv.String(item.Value), nil
	})
	if err != nil {
		if xerrors.Is(err, memcache.ErrCacheMiss) {
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
		item, err := c.opts.client.Get(key)
		if err != nil {
			return nil, err
		}

		return xconv.String(item.Value), nil
	})
	if err != nil && !xerrors.Is(err, memcache.ErrCacheMiss) {
		return cache.NewResult(nil, err)
	}

	if xerrors.Is(err, memcache.ErrCacheMiss) || val.(string) == c.opts.nilValue {
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
	if len(expiration) > 0 && expiration[0] > 0 {
		return c.opts.client.Set(&memcache.Item{
			Key:        c.AddPrefix(key),
			Value:      []byte(xconv.String(value)),
			Expiration: int32(expiration[0].Seconds()),
		})
	} else {
		return c.opts.client.Set(&memcache.Item{
			Key:   c.AddPrefix(key),
			Value: []byte(xconv.String(value)),
		})
	}
}

// GetSet 获取设置缓存值
func (c *Cache) GetSet(ctx context.Context, key string, fn cache.SetValueFunc, expiration ...time.Duration) cache.Result {
	key = c.AddPrefix(key)

	val, err, _ := c.sfg.Do(key, func() (any, error) {
		item, err := c.opts.client.Get(key)
		if err != nil {
			return nil, err
		}

		return xconv.String(item.Value), nil
	})
	if err != nil && !xerrors.Is(err, memcache.ErrCacheMiss) {
		return cache.NewResult(nil, err)
	}

	if err == nil {
		if val == c.opts.nilValue {
			return cache.NewResult(nil, xerrors.ErrNil)
		} else {
			return cache.NewResult(val)
		}
	}

	rst, _, _ := c.sfg.Do(key+":set", func() (any, error) {
		val, err := fn()
		if err != nil {
			return cache.NewResult(nil, err), nil
		}

		if val == nil || xreflect.IsNil(val) {
			if err = c.opts.client.Set(&memcache.Item{
				Key:        key,
				Value:      xconv.Bytes(c.opts.nilValue),
				Expiration: int32(c.opts.nilExpiration.Seconds()),
			}); err != nil {
				return cache.NewResult(nil, err), nil
			}
			return cache.NewResult(nil, xerrors.ErrNil), nil
		}

		expiration := c.GetExpiration(expiration...)

		if err = c.opts.client.Set(&memcache.Item{
			Key:        key,
			Value:      xconv.Bytes(val),
			Expiration: int32(expiration.Seconds()),
		}); err != nil {
			return cache.NewResult(nil, err), nil
		}

		return cache.NewResult(val, nil), nil
	})

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
	total := int64(0)

	eg, _ := errgroup.WithContext(ctx)

	for _, key := range keys {
		key = c.AddPrefix(key)

		eg.Go(func() error {
			if err := c.opts.client.Delete(key); err != nil {
				if xerrors.Is(err, memcache.ErrCacheMiss) {
					return nil
				}

				return err
			}

			atomic.AddInt64(&total, 1)

			return nil
		})
	}

	err := eg.Wait()

	if total > 0 {
		return total, nil
	}

	return 0, err
}

// IncrInt 整数自增
func (c *Cache) IncrInt(ctx context.Context, key string, value int64) (int64, error) {
	if value < 0 {
		return c.DecrInt(ctx, key, 0-value)
	}

	key = c.AddPrefix(key)

	newValue, err := c.opts.client.Increment(key, uint64(value))
	if err != nil {
		if xerrors.Is(err, memcache.ErrCacheMiss) {
			if err = c.opts.client.Add(&memcache.Item{
				Key:   key,
				Value: xconv.Bytes(xconv.String(value)),
			}); err != nil {
				if xerrors.Is(err, memcache.ErrNotStored) {
					newValue, err = c.opts.client.Increment(key, uint64(value))
					if err != nil {
						return 0, err
					}

					return int64(newValue), nil
				}
				return 0, err
			}

			return value, nil
		} else {
			return 0, err
		}
	}

	return int64(newValue), nil
}

// IncrFloat 浮点数自增，鉴于memcache不支持浮点数，所以这里是通过整数自增来实现的
func (c *Cache) IncrFloat(ctx context.Context, key string, value float64) (float64, error) {
	newValue, err := c.IncrInt(ctx, key, int64(value))
	if err != nil {
		return 0, err
	}

	return float64(newValue), nil
}

// DecrInt 整数自减
func (c *Cache) DecrInt(ctx context.Context, key string, value int64) (int64, error) {
	if value < 0 {
		return c.IncrInt(ctx, key, 0-value)
	}

	key = c.AddPrefix(key)

	newValue, err := c.opts.client.Decrement(key, uint64(value))
	if err != nil {
		if xerrors.Is(err, memcache.ErrCacheMiss) {
			if err = c.opts.client.Add(&memcache.Item{
				Key:   key,
				Value: xconv.Bytes(xconv.String(value)),
			}); err != nil {
				if xerrors.Is(err, memcache.ErrNotStored) {
					newValue, err = c.opts.client.Decrement(key, uint64(value))
					if err != nil {
						return 0, err
					}

					return int64(newValue), nil
				}
				return 0, err
			}

			return value, nil
		} else {
			return 0, err
		}
	}

	return int64(newValue), nil
}

// DecrFloat 浮点数自减，鉴于memcache不支持浮点数，所以这里是通过整数自减来实现的
func (c *Cache) DecrFloat(ctx context.Context, key string, value float64) (float64, error) {
	newValue, err := c.DecrInt(ctx, key, int64(value))
	if err != nil {
		return 0, err
	}

	return float64(newValue), nil
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

// Close 关闭客户端
func (c *Cache) Close() error {
	if !c.builtin {
		return nil
	}

	return c.opts.client.Close()
}
