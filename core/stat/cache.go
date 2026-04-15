package stat

import (
	"sync"
	"time"
)

// Cache 文件元数据缓存。
type Cache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]*cacheItem
	sf    Group
}

type cacheItem struct {
	info     FileInfo
	err      error
	expireAt int64
}

// NewCache 创建文件元数据缓存。
// ttl <= 0 表示不过期。
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl:   ttl,
		items: make(map[string]*cacheItem, 256),
	}
}

// Get 获取文件信息。
// 命中缓存且未过期时直接返回，否则重新加载。
func (c *Cache) Get(path string) (FileInfo, error) {
	if c == nil {
		return Stat(path)
	}

	now := time.Now().UnixNano()

	c.mu.RLock()
	item, ok := c.items[path]
	if ok && item != nil && (c.ttl <= 0 || item.expireAt > now) {
		info, err := item.info, item.err
		c.mu.RUnlock()
		return info, err
	}
	c.mu.RUnlock()

	v, err, _ := c.sf.Do(path, func() (any, error) {
		info, statErr := Stat(path)

		expireAt := int64(0)
		if c.ttl > 0 {
			expireAt = time.Now().Add(c.ttl).UnixNano()
		}

		c.mu.Lock()
		c.items[path] = &cacheItem{
			info:     info,
			err:      statErr,
			expireAt: expireAt,
		}
		c.mu.Unlock()

		if statErr != nil {
			return nil, statErr
		}
		return info, nil
	})
	if err != nil {
		return nil, err
	}

	info, _ := v.(FileInfo)
	return info, nil
}

// MustGet 获取文件信息，失败返回 nil。
func (c *Cache) MustGet(path string) FileInfo {
	info, err := c.Get(path)
	if err != nil {
		return nil
	}
	return info
}

// Delete 删除指定路径缓存。
func (c *Cache) Delete(path string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	delete(c.items, path)
	c.mu.Unlock()
}

// Clear 清空缓存。
func (c *Cache) Clear() {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.items = make(map[string]*cacheItem, 256)
	c.mu.Unlock()
}

// CleanupExpired 清理过期缓存。
func (c *Cache) CleanupExpired() int {
	if c == nil || c.ttl <= 0 {
		return 0
	}

	now := time.Now().UnixNano()
	n := 0

	c.mu.Lock()
	for k, v := range c.items {
		if v == nil || v.expireAt <= now {
			delete(c.items, k)
			n++
		}
	}
	c.mu.Unlock()

	return n
}

// Len 返回缓存项数量。
func (c *Cache) Len() int {
	if c == nil {
		return 0
	}

	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}
