package xcache

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

type Item struct {
	Object     interface{}
	Expiration int64
}

// Expired 判断条目是否已经过期。
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

const (
	// NoExpiration 表示条目永不过期，用于接受过期时间参数的方法。
	NoExpiration time.Duration = -1
	// DefaultExpiration 表示使用 New 或 NewFrom 创建缓存时指定的默认过期时间。
	DefaultExpiration time.Duration = 0
)

type XCache struct {
	*xcache
	// 通过外层对象管理清理协程的生命周期，详见 newCacheWithJanitor。
}

type xcache struct {
	defaultExpiration time.Duration
	items             map[string]Item
	mu                sync.RWMutex
	onEvicted         func(string, interface{})
	janitor           *janitor
}

// Set 写入条目并覆盖已有值；DefaultExpiration 使用默认过期时间，NoExpiration 表示永不过期。
func (c *xcache) Set(k string, x interface{}, d time.Duration) {
	// 直接展开 set 的逻辑，计算过期时间后加锁写入。
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.mu.Lock()
	c.items[k] = Item{
		Object:     x,
		Expiration: e,
	}
	// 此处沿用显式解锁，不使用延迟调用。
	c.mu.Unlock()
}

func (c *xcache) set(k string, x interface{}, d time.Duration) {
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	c.items[k] = Item{
		Object:     x,
		Expiration: e,
	}
}

// SetDefault 使用默认过期时间写入条目，并覆盖已有值。
func (c *xcache) SetDefault(k string, x interface{}) {
	c.Set(k, x, DefaultExpiration)
}

// Add 仅在键不存在或已有条目过期时写入，否则返回错误。
func (c *xcache) Add(k string, x interface{}, d time.Duration) error {
	c.mu.Lock()
	_, found := c.get(k)
	if found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s already exists", k)
	}
	c.set(k, x, d)
	c.mu.Unlock()
	return nil
}

// Replace 仅在键存在且条目未过期时替换其值，否则返回错误。
func (c *xcache) Replace(k string, x interface{}, d time.Duration) error {
	c.mu.Lock()
	_, found := c.get(k)
	if !found {
		c.mu.Unlock()
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	c.set(k, x, d)
	c.mu.Unlock()
	return nil
}

// Get 获取条目，返回缓存值和是否命中；不存在或已过期时返回 nil、false。
func (c *xcache) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	// 直接展开查询和过期判断逻辑。
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, false
		}
	}
	c.mu.RUnlock()
	return item.Object, true
}

// GetWithExpiration 返回缓存值、过期时间和是否命中。
// 永不过期的条目返回零值时间；不存在或已过期时返回 nil、零值时间和 false。
func (c *xcache) GetWithExpiration(k string) (interface{}, time.Time, bool) {
	c.mu.RLock()
	// 直接展开查询和过期判断逻辑。
	item, found := c.items[k]
	if !found {
		c.mu.RUnlock()
		return nil, time.Time{}, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, time.Time{}, false
		}

		// 返回缓存值和对应的过期时间。
		c.mu.RUnlock()
		return item.Object, time.Unix(0, item.Expiration), true
	}

	// 过期时间不大于零表示永不过期，返回缓存值和零值时间。
	c.mu.RUnlock()
	return item.Object, time.Time{}, true
}

func (c *xcache) get(k string) (interface{}, bool) {
	item, found := c.items[k]
	if !found {
		return nil, false
	}
	// 直接判断条目是否过期。
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}
	return item.Object, true
}

// Increment 将整数或浮点数条目的值增加 n；条目不存在、已过期或类型不支持时返回错误。
// 如需取得更新后的值，请使用 IncrementInt64 等类型专用方法。
func (c *xcache) Increment(k string, n int64) error {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Object.(type) {
	case int:
		v.Object = v.Object.(int) + int(n)
	case int8:
		v.Object = v.Object.(int8) + int8(n)
	case int16:
		v.Object = v.Object.(int16) + int16(n)
	case int32:
		v.Object = v.Object.(int32) + int32(n)
	case int64:
		v.Object = v.Object.(int64) + n
	case uint:
		v.Object = v.Object.(uint) + uint(n)
	case uintptr:
		v.Object = v.Object.(uintptr) + uintptr(n)
	case uint8:
		v.Object = v.Object.(uint8) + uint8(n)
	case uint16:
		v.Object = v.Object.(uint16) + uint16(n)
	case uint32:
		v.Object = v.Object.(uint32) + uint32(n)
	case uint64:
		v.Object = v.Object.(uint64) + uint64(n)
	case float32:
		v.Object = v.Object.(float32) + float32(n)
	case float64:
		v.Object = v.Object.(float64) + float64(n)
	default:
		c.mu.Unlock()
		return fmt.Errorf("The value for %s is not an integer", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nil
}

// IncrementFloat 将 float32 或 float64 条目的值增加 n，传入负数表示减少。
// 条目不存在、已过期或类型不匹配时返回错误；如需取得新值，请使用 IncrementFloat64 等方法。
func (c *xcache) IncrementFloat(k string, n float64) error {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Object.(type) {
	case float32:
		v.Object = v.Object.(float32) + float32(n)
	case float64:
		v.Object = v.Object.(float64) + n
	default:
		c.mu.Unlock()
		return fmt.Errorf("The value for %s does not have type float32 or float64", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nil
}

// IncrementInt 将 int 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementInt(k string, n int) (int, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementInt8 将 int8 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementInt8(k string, n int8) (int8, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int8)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int8", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementInt16 将 int16 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementInt16(k string, n int16) (int16, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int16)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int16", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementInt32 将 int32 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementInt32(k string, n int32) (int32, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int32)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int32", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementInt64 将 int64 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementInt64(k string, n int64) (int64, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int64)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int64", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementUint 将 uint 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementUint(k string, n uint) (uint, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementUintptr 将 uintptr 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementUintptr(k string, n uintptr) (uintptr, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uintptr)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uintptr", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementUint8 将 uint8 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementUint8(k string, n uint8) (uint8, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint8)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint8", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementUint16 将 uint16 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementUint16(k string, n uint16) (uint16, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint16)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint16", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementUint32 将 uint32 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementUint32(k string, n uint32) (uint32, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint32)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint32", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementUint64 将 uint64 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementUint64(k string, n uint64) (uint64, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint64)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint64", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementFloat32 将 float32 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementFloat32(k string, n float32) (float32, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float32)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float32", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// IncrementFloat64 将 float64 类型条目的值增加 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) IncrementFloat64(k string, n float64) (float64, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float64)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float64", k)
	}
	nv := rv + n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// Decrement 将整数或浮点数条目的值减少 n；条目不存在、已过期或类型不支持时返回错误。
// 如需取得更新后的值，请使用 DecrementInt64 等类型专用方法。
func (c *xcache) Decrement(k string, n int64) error {
	// 待优化：整理 Increment 和 Decrement 的实现；无符号整数不能简单通过 Increment(k, n*-1) 处理。
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return fmt.Errorf("Item not found")
	}
	switch v.Object.(type) {
	case int:
		v.Object = v.Object.(int) - int(n)
	case int8:
		v.Object = v.Object.(int8) - int8(n)
	case int16:
		v.Object = v.Object.(int16) - int16(n)
	case int32:
		v.Object = v.Object.(int32) - int32(n)
	case int64:
		v.Object = v.Object.(int64) - n
	case uint:
		v.Object = v.Object.(uint) - uint(n)
	case uintptr:
		v.Object = v.Object.(uintptr) - uintptr(n)
	case uint8:
		v.Object = v.Object.(uint8) - uint8(n)
	case uint16:
		v.Object = v.Object.(uint16) - uint16(n)
	case uint32:
		v.Object = v.Object.(uint32) - uint32(n)
	case uint64:
		v.Object = v.Object.(uint64) - uint64(n)
	case float32:
		v.Object = v.Object.(float32) - float32(n)
	case float64:
		v.Object = v.Object.(float64) - float64(n)
	default:
		c.mu.Unlock()
		return fmt.Errorf("The value for %s is not an integer", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nil
}

// DecrementFloat 将 float32 或 float64 条目的值减少 n，传入负数表示增加。
// 条目不存在、已过期或类型不匹配时返回错误；如需取得新值，请使用 DecrementFloat64 等方法。
func (c *xcache) DecrementFloat(k string, n float64) error {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Object.(type) {
	case float32:
		v.Object = v.Object.(float32) - float32(n)
	case float64:
		v.Object = v.Object.(float64) - n
	default:
		c.mu.Unlock()
		return fmt.Errorf("The value for %s does not have type float32 or float64", k)
	}
	c.items[k] = v
	c.mu.Unlock()
	return nil
}

// DecrementInt 将 int 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementInt(k string, n int) (int, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementInt8 将 int8 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementInt8(k string, n int8) (int8, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int8)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int8", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementInt16 将 int16 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementInt16(k string, n int16) (int16, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int16)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int16", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementInt32 将 int32 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementInt32(k string, n int32) (int32, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int32)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int32", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementInt64 将 int64 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementInt64(k string, n int64) (int64, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int64)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int64", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementUint 将 uint 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementUint(k string, n uint) (uint, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementUintptr 将 uintptr 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementUintptr(k string, n uintptr) (uintptr, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uintptr)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uintptr", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementUint8 将 uint8 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementUint8(k string, n uint8) (uint8, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint8)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint8", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementUint16 将 uint16 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementUint16(k string, n uint16) (uint16, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint16)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint16", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementUint32 将 uint32 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementUint32(k string, n uint32) (uint32, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint32)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint32", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementUint64 将 uint64 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementUint64(k string, n uint64) (uint64, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint64)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint64", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementFloat32 将 float32 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementFloat32(k string, n float32) (float32, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float32)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float32", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// DecrementFloat64 将 float64 类型条目的值减少 n，并返回更新后的值。
// 条目不存在、已过期或类型不匹配时返回错误。
func (c *xcache) DecrementFloat64(k string, n float64) (float64, error) {
	c.mu.Lock()
	v, found := c.items[k]
	if !found || v.Expired() {
		c.mu.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float64)
	if !ok {
		c.mu.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float64", k)
	}
	nv := rv - n
	v.Object = nv
	c.items[k] = v
	c.mu.Unlock()
	return nv, nil
}

// Delete 删除指定条目；键不存在时不执行操作。
func (c *xcache) Delete(k string) {
	c.mu.Lock()
	v, evicted := c.delete(k)
	callback := c.onEvicted
	c.mu.Unlock()
	if evicted {
		callback(k, v)
	}
}

func (c *xcache) delete(k string) (interface{}, bool) {
	if c.onEvicted != nil {
		if v, found := c.items[k]; found {
			delete(c.items, k)
			return v.Object, true
		}
	}
	delete(c.items, k)
	return nil, false
}

type keyAndValue struct {
	key   string
	value interface{}
}

// DeleteExpired 删除所有已过期条目，并在锁外执行删除回调。
func (c *xcache) DeleteExpired() {
	var evictedItems []keyAndValue
	now := time.Now().UnixNano()
	c.mu.Lock()
	for k, v := range c.items {
		// 直接判断条目是否过期。
		if v.Expiration > 0 && now > v.Expiration {
			ov, evicted := c.delete(k)
			if evicted {
				evictedItems = append(evictedItems, keyAndValue{k, ov})
			}
		}
	}
	callback := c.onEvicted
	c.mu.Unlock()
	for _, v := range evictedItems {
		callback(v.key, v.value)
	}
}

// OnEvicted 设置条目被删除时的可选回调，参数为键和缓存值。
// 手动删除和过期清理会触发回调，覆盖不会触发；传入 nil 可禁用后续删除的回调。
func (c *xcache) OnEvicted(f func(string, interface{})) {
	c.mu.Lock()
	c.onEvicted = f
	c.mu.Unlock()
}

// Save 使用 Gob 将缓存条目写入指定输出流。
// 此方法已弃用，建议使用 Items 和 NewFrom，参见 NewFrom 的说明。
func (c *xcache) Save(w io.Writer) (err error) {
	enc := gob.NewEncoder(w)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Error registering item types with Gob library")
		}
	}()
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, v := range c.items {
		gob.Register(v.Object)
	}
	err = enc.Encode(&c.items)
	return
}

// SaveFile 将缓存条目保存到指定文件；文件不存在时创建，存在时覆盖。
// 此方法已弃用，建议使用 Items 和 NewFrom，参见 NewFrom 的说明。
func (c *xcache) SaveFile(fname string) error {
	fp, err := os.Create(fname)
	if err != nil {
		return err
	}
	err = c.Save(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// Load 从输入流读取 Gob 编码的缓存条目，不覆盖当前缓存中仍未过期的同名条目。
// 此方法已弃用，建议使用 Items 和 NewFrom，参见 NewFrom 的说明。
func (c *xcache) Load(r io.Reader) error {
	dec := gob.NewDecoder(r)
	items := map[string]Item{}
	err := dec.Decode(&items)
	if err == nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k, v := range items {
			ov, found := c.items[k]
			if !found || ov.Expired() {
				c.items[k] = v
			}
		}
	}
	return err
}

// LoadFile 从指定文件加载缓存条目，不覆盖当前缓存中仍未过期的同名条目。
// 此方法已弃用，建议使用 Items 和 NewFrom，参见 NewFrom 的说明。
func (c *xcache) LoadFile(fname string) error {
	fp, err := os.Open(fname)
	if err != nil {
		return err
	}
	err = c.Load(fp)
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// Items 将所有未过期条目浅拷贝到新的 map 并返回。
func (c *xcache) Items() map[string]Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[string]Item, len(c.items))
	now := time.Now().UnixNano()
	for k, v := range c.items {
		// 直接判断条目是否过期。
		if v.Expiration > 0 {
			if now > v.Expiration {
				continue
			}
		}
		m[k] = v
	}
	return m
}

// ItemCount 返回内部条目数量，可能包含已经过期但尚未清理的条目。
func (c *xcache) ItemCount() int {
	c.mu.RLock()
	n := len(c.items)
	c.mu.RUnlock()
	return n
}

// Flush 清空所有条目，不触发删除回调。
func (c *xcache) Flush() {
	c.mu.Lock()
	c.items = map[string]Item{}
	c.mu.Unlock()
}

type janitor struct {
	Interval time.Duration
	stop     chan struct{}
	stopOnce sync.Once
}

func (j *janitor) Run(c *xcache) {
	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			return
		}
	}
}

func stopJanitor(c *XCache) {
	c.Close()
}

// Close 停止后台清理，可重复或并发调用；不清空数据，已有缓存仍可读写。
// 此方法不等待正在执行的清理和回调结束，回调内调用也不会死锁。
func (c *XCache) Close() {
	if c.janitor != nil {
		c.janitor.stopOnce.Do(func() { close(c.janitor.stop) })
	}
}

func runJanitor(c *xcache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
		stop:     make(chan struct{}),
	}
	c.janitor = j
	go j.Run(c)
}

func newCache(de time.Duration, m map[string]Item) *xcache {
	if m == nil {
		m = make(map[string]Item)
	}
	if de == 0 {
		de = -1
	}
	c := &xcache{
		defaultExpiration: de,
		items:             m,
	}
	return c
}

func newCacheWithJanitor(de time.Duration, ci time.Duration, m map[string]Item) *XCache {
	c := newCache(de, m)
	// 清理协程只持有内部缓存，不持有返回的外层对象。
	// 外层对象被垃圾回收时，通过终结器停止清理协程，使内部缓存也能被回收。
	C := &XCache{c}
	if ci > 0 {
		runJanitor(c, ci)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C
}

// New 创建缓存并指定默认过期时间和后台清理间隔。
// 默认过期时间不大于零表示默认永不过期，需要手动删除。
// 清理间隔不大于零表示不启动后台清理，可通过 DeleteExpired 主动清理过期条目。
func New(defaultExpiration, cleanupInterval time.Duration) *XCache {
	items := make(map[string]Item)
	return newCacheWithJanitor(defaultExpiration, cleanupInterval, items)
}

// NewFrom 从已有数据创建缓存，复制 map，避免创建后与调用方共享底层 map。
// 构造期间调用方不能并发修改输入 map；Object 中的指针、切片等仍是浅拷贝。
// 默认过期时间不大于零表示不过期，清理间隔不大于零表示关闭后台清理。
// 输入条目保留原来的绝对过期时间；nil map 等同于空缓存。
func NewFrom(defaultExpiration, cleanupInterval time.Duration, items map[string]Item) *XCache {
	copied := make(map[string]Item, len(items))
	for key, item := range items {
		copied[key] = item
	}
	return newCacheWithJanitor(defaultExpiration, cleanupInterval, copied)
}
