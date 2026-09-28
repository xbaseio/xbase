package xcache

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewFromOwnsMap 验证空输入可写，且构造后不再共享调用方的 map。
func TestNewFromOwnsMap(t *testing.T) {
	cache := NewFrom(time.Minute, 0, nil)
	cache.SetDefault("key", "value")
	if value, found := cache.Get("key"); !found || value != "value" {
		t.Fatalf("空输入构造失败: %v %v", value, found)
	}
	items := map[string]Item{"key": {Object: "original"}}
	cache = NewFrom(time.Minute, 0, items)
	items["key"] = Item{Object: "changed"}
	if value, _ := cache.Get("key"); value != "original" {
		t.Fatalf("缓存仍共享输入 map: %v", value)
	}
	cache.SetDefault("other", "value")
	if _, found := items["other"]; found {
		t.Fatal("缓存写入影响了输入 map")
	}
}

// TestEvictionCallbackSnapshot 验证回调可重入，并使用删除时确定的回调完成整批通知。
func TestEvictionCallbackSnapshot(t *testing.T) {
	cache := NewFrom(time.Minute, 0, map[string]Item{
		"first":  {Object: 1, Expiration: time.Now().Add(-time.Second).UnixNano()},
		"second": {Object: 2, Expiration: time.Now().Add(-time.Second).UnixNano()},
	})
	calls := 0
	cache.OnEvicted(func(key string, value any) {
		calls++
		cache.OnEvicted(nil)
		cache.SetDefault("replacement", value)
	})
	cache.DeleteExpired()
	if calls != 2 {
		t.Fatalf("删除回调次数: %d", calls)
	}
	cache.OnEvicted(func(key string, value any) {
		cache.SetDefault("restored", value)
	})
	cache.Delete("replacement")
	if _, found := cache.Get("restored"); !found {
		t.Fatal("删除回调未执行")
	}
}

// TestConcurrentEvictionCallback 验证注册回调与删除、清理并发执行时没有竞态。
func TestConcurrentEvictionCallback(t *testing.T) {
	cache := New(time.Minute, 0)
	var workers sync.WaitGroup
	workers.Add(3)
	go func() {
		defer workers.Done()
		for range 1000 {
			cache.OnEvicted(func(string, any) {})
			cache.OnEvicted(nil)
		}
	}()
	go func() {
		defer workers.Done()
		for range 1000 {
			cache.SetDefault("delete", "value")
			cache.Delete("delete")
		}
	}()
	go func() {
		defer workers.Done()
		for range 1000 {
			cache.Set("expire", "value", time.Nanosecond)
			cache.DeleteExpired()
		}
	}()
	workers.Wait()
}

// TestCloseJanitor 验证清理协程停止信号可立即、重复及并发发送。
func TestCloseJanitor(t *testing.T) {
	cache := New(time.Minute, time.Hour)
	sharded := unexportedNewSharded(time.Minute, time.Hour, 4)
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			cache.Close()
			sharded.Close()
		})
	}
	workers.Wait()
	select {
	case <-cache.janitor.stop:
	default:
		t.Fatal("普通缓存未收到停止信号")
	}
	select {
	case <-sharded.janitor.stop:
	default:
		t.Fatal("分片缓存未收到停止信号")
	}
	cache.SetDefault("after-close", "value")
	if _, found := cache.Get("after-close"); !found {
		t.Fatal("停止清理不应禁止缓存访问")
	}
	New(time.Minute, 0).Close()
	unexportedNewSharded(time.Minute, 0, 4).Close()
}

// TestHashIncludesLastByte 验证不同长度键的最后一个字节参与哈希。
func TestHashIncludesLastByte(t *testing.T) {
	for length := range 16 {
		prefix := strings.Repeat("a", length)
		if djb33(123, prefix+"x") == djb33(123, prefix+"y") {
			t.Fatalf("末尾字符未参与哈希，前缀长度: %d", length)
		}
	}
}

// TestShardsMustBePositive 验证非法分片数在构造阶段被拒绝。
func TestShardsMustBePositive(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("零分片应在构造时被拒绝")
		}
	}()
	unexportedNewSharded(time.Minute, 0, 0)
}
