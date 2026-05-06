package xrand

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	LetterSeed           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	LetterLowerSeed      = "abcdefghijklmnopqrstuvwxyz"
	LetterUpperSeed      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DigitSeed            = "0123456789"
	DigitWithoutZeroSeed = "123456789"
	SymbolSeed           = "!\\\"#$%&'()*+,-./:;<=>?@[\\\\]^_`{|}~"
	Base62Seed           = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	HumanSafeSeed        = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// 熵池大小：4KB。
// 够大，能显著减少 crypto/rand 调用频率；
// 又不至于太大导致浪费。
const entropyChunkSize = 4096

// 黄金比例常数
const goldenRatio64 = uint64(0x9e3779b97f4a7c15)

// entropyChunk 是当前熵块。
// gen 用于配合 cursor 做无锁切换。
type entropyChunk struct {
	gen uint32
	buf [entropyChunkSize]byte
}

var (
	currentChunk atomic.Pointer[entropyChunk]
	cursor       atomic.Uint64 // 高32位: gen, 低32位: offset
	refillState  atomic.Uint32 // 0=idle, 1=refilling
	seedCounter  atomic.Uint64

	randPool = sync.Pool{
		New: func() any {
			return rand.New(rand.NewSource(newSeed()))
		},
	}
)

func init() {
	_ = refillEntropy()
}

// Rand 返回一个新的 *rand.Rand。
// 注意：返回的是独立实例，调用方自己持有使用。
func Rand() *rand.Rand {
	return rand.New(rand.NewSource(newSeed()))
}

// HumanSafeRand 返回一个新的 *rand.Rand，使用人类友好的种子字符集。
func HumanSafeSeedRand(length int) string {
	return Str(HumanSafeSeed, length)
}

// Str 生成指定长度的字符串。
// 优先使用安全随机；若安全随机不可用，则回退到 math/rand。
func Str(seed string, length int) string {
	if length <= 0 {
		return ""
	}
	r := []rune(seed)
	n := len(r)
	if n == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(length)

	for i := 0; i < length; i++ {
		pos := intnHybrid(n)
		b.WriteRune(r[pos])
	}
	return b.String()
}

// SecureStr 强制优先使用安全随机；若安全熵池暂时不可用，也会自动回退。
// 这里保留接口语义，内部仍然是混合模式。
func SecureStr(seed string, length int) string {
	return Str(seed, length)
}

// FastStr 纯 math/rand，高性能但不安全。
func FastStr(seed string, length int) string {
	if length <= 0 {
		return ""
	}
	r := []rune(seed)
	n := len(r)
	if n == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(length)

	for i := 0; i < length; i++ {
		b.WriteRune(r[fastIntn(n)])
	}
	return b.String()
}

// Base62Rand 生成 base62 随机串。
func Base62Rand(length int) string {
	return Str(Base62Seed, length)
}

// Letters 生成指定长度的字母字符串。
func Letters(length int) string {
	return Str(LetterSeed, length)
}

// Digits 生成指定长度的数字字符串。
// 默认首位不为 0；传 true 则允许首位为 0。
func Digits(length int, hasLeadingZero ...bool) string {
	if length <= 0 {
		return ""
	}

	if len(hasLeadingZero) > 0 && hasLeadingZero[0] {
		return Str(DigitSeed, length)
	}

	if length == 1 {
		return Str(DigitWithoutZeroSeed, 1)
	}

	return Str(DigitWithoutZeroSeed, 1) + Str(DigitSeed, length-1)
}

// Symbols 生成指定长度的特殊字符字符串。
func Symbols(length int) string {
	return Str(SymbolSeed, length)
}

// Int 生成 [min,max] 的整数。
func Int(min, max int) int {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}

	span := max - min + 1
	if span <= 0 {
		// 理论上只有 int 溢出时才会到这里，兜底走 fast。
		return min + fastIntn(max-min+1)
	}
	return min + intnHybrid(span)
}

// Int32 生成 [min,max] 的 int32。
func Int32(min, max int32) int32 {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}

	span := int64(max) - int64(min) + 1
	return min + int32(int63nHybrid(span))
}

// Int64 生成 [min,max] 的 int64。
func Int64(min, max int64) int64 {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}

	span := max - min + 1
	if span <= 0 {
		// 极端溢出保护
		return min + fastInt63n(max-min+1)
	}
	return min + int63nHybrid(span)
}

// Float32 生成 [min,max) 的 float32。
func Float32(min, max float32) float32 {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	return min + float32(float64(max-min)*float64(hybridFloat64()))
}

// Float64 生成 [min,max) 的 float64。
func Float64(min, max float64) float64 {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	return min + (max-min)*hybridFloat64()
}

// Duration 生成 [min,max] 的时间间隔。
func Duration(min, max time.Duration) time.Duration {
	return time.Duration(Int64(int64(min), int64(max)))
}

// Lucky 根据概率抽取幸运值。
// 默认基数 100，可传 base 覆盖。
// 例如：Lucky(12.5) 表示 12.5%。
func Lucky(probability float64, base ...float64) bool {
	if probability <= 0 {
		return false
	}

	b := 100.0
	if len(base) > 0 && base[0] > 0 {
		b = base[0]
	}

	if probability >= b {
		return true
	}

	str := strconv.FormatFloat(probability, 'f', -1, 64)
	scale := float64(1)

	if i := strings.IndexByte(str, '.'); i > 0 {
		scale = math.Pow10(len(str) - i - 1)
	}

	maxN := int64(b * scale)
	cur := int64(probability * scale)

	if maxN <= 0 {
		return false
	}
	return Int64(1, maxN) <= cur
}

// Weight 权重随机，返回命中的下标。
// 返回 -1 表示 list 为空。
func Weight(fn func(v any) float64, list ...any) int {
	if len(list) == 0 {
		return -1
	}

	total := 0.0
	scale := 1.0

	for _, item := range list {
		weight := fn(item)
		if weight <= 0 {
			continue
		}

		str := strconv.FormatFloat(weight, 'f', -1, 64)
		if i := strings.IndexByte(str, '.'); i > 0 {
			scale = math.Max(scale, math.Pow10(len(str)-i-1))
		}

		total += weight
	}

	sum := int64(total * scale)
	if sum <= 0 {
		return Int(0, len(list)-1)
	}

	pick := Int64(1, sum)
	acc := int64(0)

	for i, item := range list {
		w := fn(item)
		if w <= 0 {
			continue
		}
		acc += int64(w * scale)
		if pick <= acc {
			return i
		}
	}

	return len(list) - 1
}

// Shuffle 打乱切片。
func Shuffle[T any](list []T) {
	if len(list) <= 1 {
		return
	}
	withFastRand(func(r *rand.Rand) {
		r.Shuffle(len(list), func(i, j int) {
			list[i], list[j] = list[j], list[i]
		})
	})
}

// -----------------------------------------------------------------------------
// 混合随机核心
// -----------------------------------------------------------------------------

func newSeed() int64 {
	// 混合时间 + 原子计数，避免并发下重复 seed。

	now := uint64(time.Now().UnixNano())
	seq := seedCounter.Add(goldenRatio64)
	return int64(now ^ seq)
}

func withFastRand(fn func(r *rand.Rand)) {
	r := randPool.Get().(*rand.Rand)
	fn(r)
	randPool.Put(r)
}

func fastIntn(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	var v int
	withFastRand(func(r *rand.Rand) {
		v = r.Intn(n)
	})
	return v
}

func fastInt63n(n int64) int64 {
	if n <= 0 {
		panic("invalid argument to Int63n")
	}
	var v int64
	withFastRand(func(r *rand.Rand) {
		v = r.Int63n(n)
	})
	return v
}

func fastUint64() uint64 {
	var v uint64
	withFastRand(func(r *rand.Rand) {
		hi := uint64(r.Uint32())
		lo := uint64(r.Uint32())
		v = (hi << 32) | lo
	})
	return v
}

func fastFloat64() float64 {
	var v float64
	withFastRand(func(r *rand.Rand) {
		v = r.Float64()
	})
	return v
}

func intnHybrid(n int) int {
	if n <= 0 {
		panic("invalid argument to Intn")
	}
	if v, ok := secureIntn(n); ok {
		return v
	}
	return fastIntn(n)
}

func int63nHybrid(n int64) int64 {
	if n <= 0 {
		panic("invalid argument to Int63n")
	}
	if v, ok := secureInt63n(n); ok {
		return v
	}
	return fastInt63n(n)
}

func hybridFloat64() float64 {
	if v, ok := secureFloat64(); ok {
		return v
	}
	return fastFloat64()
}

func secureIntn(n int) (int, bool) {
	v, ok := secureInt63n(int64(n))
	return int(v), ok
}

func secureInt63n(n int64) (int64, bool) {
	if n <= 0 {
		return 0, false
	}

	// rejection sampling，避免取模偏差
	limit := uint64(math.MaxUint64 - (math.MaxUint64 % uint64(n)))
	for {
		v, ok := secureUint64()
		if !ok {
			return 0, false
		}
		if v < limit {
			return int64(v % uint64(n)), true
		}
	}
}

func secureFloat64() (float64, bool) {
	// 取 53 bit，和 math/rand.Float64 一样的精度思路
	v, ok := secureUint64()
	if !ok {
		return 0, false
	}
	return float64(v>>11) / (1 << 53), true
}

func secureUint64() (uint64, bool) {
	var b [8]byte
	for i := 0; i < 8; i++ {
		v, ok := secureByte()
		if !ok {
			return 0, false
		}
		b[i] = v
	}
	return binary.LittleEndian.Uint64(b[:]), true
}

// secureByte：
// 快路径：CAS 从当前熵块中取字节，基本无锁。
// 慢路径：熵块耗尽时，由一个 goroutine 负责 refill，其他 goroutine 自旋让出。
func secureByte() (byte, bool) {
	for {
		ch := currentChunk.Load()
		if ch == nil {
			if err := refillEntropy(); err != nil {
				return 0, false
			}
			continue
		}

		cur := cursor.Load()
		gen := uint32(cur >> 32)
		off := uint32(cur)

		// 代际不一致，说明熵块已切换，重试
		if gen != ch.gen {
			_ = cursor.CompareAndSwap(cur, packCursor(ch.gen, 0))
			continue
		}

		if off >= entropyChunkSize {
			if err := refillEntropy(); err != nil {
				return 0, false
			}
			continue
		}

		next := packCursor(gen, off+1)
		if cursor.CompareAndSwap(cur, next) {
			return ch.buf[off], true
		}
	}
}

func packCursor(gen, off uint32) uint64 {
	return (uint64(gen) << 32) | uint64(off)
}

func refillEntropy() error {
	// 抢到 refill 权限的 goroutine 负责填充新熵块
	if refillState.CompareAndSwap(0, 1) {
		defer refillState.Store(0)

		old := currentChunk.Load()
		nextGen := uint32(1)
		if old != nil {
			nextGen = old.gen + 1
			if nextGen == 0 {
				nextGen = 1
			}
		}

		ch := &entropyChunk{gen: nextGen}
		if _, err := crand.Read(ch.buf[:]); err != nil {
			return err
		}

		currentChunk.Store(ch)
		cursor.Store(packCursor(ch.gen, 0))
		return nil
	}

	// 其他 goroutine 等待当前 refill 完成。
	for i := 0; i < 64; i++ {
		if refillState.Load() == 0 && currentChunk.Load() != nil {
			return nil
		}
		time.Sleep(time.Microsecond)
	}

	// 最后再检查一次
	if currentChunk.Load() != nil {
		return nil
	}
	return errors.New("xrand: entropy refill timeout")
}
