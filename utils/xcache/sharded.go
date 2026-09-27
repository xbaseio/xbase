package xcache

import (
	"crypto/rand"
	"math"
	"math/big"
	insecurerand "math/rand"
	"os"
	"runtime"
	"sync"
	"time"
)

// 这是尚未导出的实验性分片缓存，写入时只锁定目标分片，避免锁住整个缓存。
// 分片选择本身也有开销，是否更快需结合数据规模和并发情况评估。
// 性能测试参见 cache_test.go 和 sharded_test.go。

type unexportedShardedCache struct {
	*shardedCache
}

type shardedCache struct {
	seed    uint32
	m       uint32
	cs      []*xcache
	janitor *shardedJanitor
}

// djb33 计算完整键的哈希，包括最后一个字节，避免相同前缀的键集中到同一分片。
func djb33(seed uint32, k string) uint32 {
	hash := uint32(5381) + seed + uint32(len(k))
	for index := 0; index < len(k); index++ {
		hash = (hash * 33) ^ uint32(k[index])
	}
	return hash ^ (hash >> 16)
}

func (sc *shardedCache) bucket(k string) *xcache {
	return sc.cs[djb33(sc.seed, k)%sc.m]
}

func (sc *shardedCache) Set(k string, x interface{}, d time.Duration) {
	sc.bucket(k).Set(k, x, d)
}

func (sc *shardedCache) Add(k string, x interface{}, d time.Duration) error {
	return sc.bucket(k).Add(k, x, d)
}

func (sc *shardedCache) Replace(k string, x interface{}, d time.Duration) error {
	return sc.bucket(k).Replace(k, x, d)
}

func (sc *shardedCache) Get(k string) (interface{}, bool) {
	return sc.bucket(k).Get(k)
}

func (sc *shardedCache) Increment(k string, n int64) error {
	return sc.bucket(k).Increment(k, n)
}

func (sc *shardedCache) IncrementFloat(k string, n float64) error {
	return sc.bucket(k).IncrementFloat(k, n)
}

func (sc *shardedCache) Decrement(k string, n int64) error {
	return sc.bucket(k).Decrement(k, n)
}

func (sc *shardedCache) Delete(k string) {
	sc.bucket(k).Delete(k)
}

func (sc *shardedCache) DeleteExpired() {
	for _, v := range sc.cs {
		v.DeleteExpired()
	}
}

// Items 返回各分片中未过期条目的浅拷贝，不与缓存共享底层 map。
// 各分片分别读取，因此结果不是所有分片在同一时刻的一致快照。
func (sc *shardedCache) Items() []map[string]Item {
	res := make([]map[string]Item, len(sc.cs))
	for i, v := range sc.cs {
		res[i] = v.Items()
	}
	return res
}

func (sc *shardedCache) Flush() {
	for _, v := range sc.cs {
		v.Flush()
	}
}

type shardedJanitor struct {
	Interval time.Duration
	stop     chan struct{}
	stopOnce sync.Once
}

func (j *shardedJanitor) Run(sc *shardedCache) {
	ticker := time.NewTicker(j.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sc.DeleteExpired()
		case <-j.stop:
			return
		}
	}
}

func stopShardedJanitor(sc *unexportedShardedCache) {
	sc.Close()
}

// Close 幂等停止分片缓存的后台清理，不等待已经开始的清理和回调。
func (sc *unexportedShardedCache) Close() {
	if sc.janitor != nil {
		sc.janitor.stopOnce.Do(func() { close(sc.janitor.stop) })
	}
}

func runShardedJanitor(sc *shardedCache, ci time.Duration) {
	j := &shardedJanitor{
		Interval: ci,
		stop:     make(chan struct{}),
	}
	sc.janitor = j
	go j.Run(sc)
}

func newShardedCache(n int, de time.Duration) *shardedCache {
	if n <= 0 {
		panic("xcache: 分片数量必须大于零")
	}
	max := big.NewInt(0).SetUint64(uint64(math.MaxUint32))
	rnd, err := rand.Int(rand.Reader, max)
	var seed uint32
	if err != nil {
		os.Stderr.Write([]byte("WARNING: go-xcache's newShardedCache failed to read from the system CSPRNG (/dev/urandom or equivalent.) Your system's security may be compromised. Continuing with an insecure seed.\n"))
		seed = insecurerand.Uint32()
	} else {
		seed = uint32(rnd.Uint64())
	}
	sc := &shardedCache{
		seed: seed,
		m:    uint32(n),
		cs:   make([]*xcache, n),
	}
	for i := 0; i < n; i++ {
		c := &xcache{
			defaultExpiration: de,
			items:             map[string]Item{},
		}
		sc.cs[i] = c
	}
	return sc
}

func unexportedNewSharded(defaultExpiration, cleanupInterval time.Duration, shards int) *unexportedShardedCache {
	if defaultExpiration == 0 {
		defaultExpiration = -1
	}
	sc := newShardedCache(shards, defaultExpiration)
	SC := &unexportedShardedCache{sc}
	if cleanupInterval > 0 {
		runShardedJanitor(sc, cleanupInterval)
		runtime.SetFinalizer(SC, stopShardedJanitor)
	}
	return SC
}
