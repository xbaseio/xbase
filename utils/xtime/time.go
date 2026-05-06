package xtime

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/etc"
)

const (
	Layout      = time.Layout
	ANSIC       = time.ANSIC
	UnixDate    = time.UnixDate
	RubyDate    = time.RubyDate
	RFC822      = time.RFC822
	RFC822Z     = time.RFC822Z
	RFC850      = time.RFC850
	RFC1123     = time.RFC1123
	RFC1123Z    = time.RFC1123Z
	RFC3339     = time.RFC3339
	RFC3339Nano = time.RFC3339Nano
	Kitchen     = time.Kitchen

	Stamp      = time.Stamp
	StampMilli = time.StampMilli
	StampMicro = time.StampMicro
	StampNano  = time.StampNano

	DateTime       = time.DateTime // 2006-01-02 15:04:05
	DateTimeLayout = time.DateTime // 2006-01-02 15:04:05
	DateOnly       = time.DateOnly // 2006-01-02
	TimeOnly       = time.TimeOnly // 15:04:05
	MonthOnly      = "2006-01"
	YearOnly       = "2006"
)

// 兼容常用命名：Go 的 time.Format 必须使用 2006-01-02 这种 layout。
const (
	TimeFormat     = TimeOnly
	DateFormat     = DateOnly
	DatetimeFormat = DateTime

	TimeLayout     = TimeOnly
	DateLayout     = DateOnly
	DatetimeLayout = DateTime
)

var (
	locationValue atomic.Value // *time.Location

	defaultTransformRule = []TransformRule{
		{Max: 60, Tpl: "刚刚"},
		{Max: 3600, Tpl: "%d分钟前"},
		{Max: 86400, Tpl: "%d小时前"},
		{Max: 2592000, Tpl: "%d天前"},
		{Max: 31536000, Tpl: "%d个月前"},
		{Max: 0, Tpl: "%d年前"},
	}
)

type TransformRule struct {
	Max uint
	Tpl string
}

type Time = time.Time

func init() {
	locationValue.Store(time.Local)

	timezone := etc.Get("etc.timezone", "Local").String()
	if err := SetTimezone(timezone); err != nil {
		locationValue.Store(time.Local)
	}
}

// Location 当前默认时区
func Location() *time.Location {
	v := locationValue.Load()
	if loc, ok := v.(*time.Location); ok && loc != nil {
		return loc
	}
	return time.Local
}

// SetLocation 设置默认时区
func SetLocation(loc *time.Location) {
	if loc == nil {
		return
	}
	locationValue.Store(loc)
}

// SetTimezone 按名称设置默认时区，例如 Asia/Shanghai、Asia/Singapore、Local
func SetTimezone(name string) error {
	if name == "" || name == "Local" {
		SetLocation(time.Local)
		return nil
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return err
	}

	SetLocation(loc)
	return nil
}

// Parse 解析日期时间
func Parse(layout string, value string) (Time, error) {
	return time.ParseInLocation(layout, value, Location())
}

// Now 当前时间
func Now() Time {
	return time.Now().In(Location())
}

// Today 今天当前时刻
func Today() Time {
	return Now()
}

// Yesterday 昨天当前时刻
func Yesterday() Time {
	return Day(-1)
}

// Tomorrow 明天当前时刻
func Tomorrow() Time {
	return Day(1)
}

// Transform 时间转换成“刚刚/几分钟前/几小时前”
// rule 可自定义，但 Max 必须从小到大，最后一项 Max 可以为 0 表示兜底。
func Transform(t Time, rule ...[]TransformRule) string {
	rules := defaultTransformRule
	if len(rule) > 0 && len(rule[0]) > 0 {
		rules = rule[0]
	}

	now := Now()
	target := t.In(Location())

	// 未来时间不做 uint 转换，避免负数溢出成超大值。
	if !target.Before(now) {
		return formatTransformTpl(rules[0].Tpl, 0)
	}

	durSec := uint64(now.Sub(target) / time.Second)
	unit := uint64(1)

	for i, r := range rules {
		if i == len(rules)-1 || r.Max == 0 || durSec < uint64(r.Max) {
			return formatTransformTpl(r.Tpl, durSec/unit)
		}

		unit = uint64(r.Max)
	}

	return ""
}

func formatTransformTpl(tpl string, v uint64) string {
	if strings.Contains(tpl, "%") {
		return fmt.Sprintf(tpl, v)
	}
	return tpl
}

// Unix 秒时间戳转标准时间
func Unix(sec int64, nsec ...int64) Time {
	ns := int64(0)
	if len(nsec) > 0 {
		ns = nsec[0]
	}
	return time.Unix(sec, ns).In(Location())
}

// UnixMilli 毫秒时间戳转标准时间
func UnixMilli(msec int64) Time {
	return time.UnixMilli(msec).In(Location())
}

// UnixMicro 微秒时间戳转标准时间
func UnixMicro(usec int64) Time {
	return time.UnixMicro(usec).In(Location())
}

// UnixNano 纳秒时间戳转标准时间
func UnixNano(nsec int64) Time {
	return time.Unix(0, nsec).In(Location())
}

// Day 获取某一天的当前时刻
// offsetDays 偏移天数，例如：-1 前一天，0 当前，1 明天
func Day(offset ...int) Time {
	return Now().AddDate(0, 0, firstOffset(offset))
}

// DayHead 获取一天开始时间
func DayHead(offset ...int) Time {
	return StartOfDay(Day(offset...))
}

// DayTail 获取一天结束时间
func DayTail(offset ...int) Time {
	return DayHead(offset...).AddDate(0, 0, 1).Add(-time.Nanosecond)
}

// Week 获取某一周的当前时刻
// offsetWeeks 偏移周数，例如：-1 上周，0 本周，1 下周
func Week(offset ...int) Time {
	return Now().AddDate(0, 0, firstOffset(offset)*7)
}

// WeekHead 获取一周开始时间，默认周一为第一天
func WeekHead(offset ...int) Time {
	base := DayHead()

	weekday := int(base.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday
	}

	offsetDays := 1 - weekday + firstOffset(offset)*7
	return base.AddDate(0, 0, offsetDays)
}

// WeekTail 获取一周结束时间，默认周日为最后一天
func WeekTail(offset ...int) Time {
	return WeekHead(offset...).AddDate(0, 0, 7).Add(-time.Nanosecond)
}

// Month 获取某一月的当前时刻
// offsetMonths 偏移月数，例如：-1 前一月，0 当前月，1 下一月
func Month(offset ...int) Time {
	return addMonthsClamped(Now(), firstOffset(offset))
}

// MonthHead 获取一月开始时间
func MonthHead(offset ...int) Time {
	return StartOfMonth(Month(offset...))
}

// MonthTail 获取一月结束时间
func MonthTail(offset ...int) Time {
	return MonthHead(offset...).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// StartOfDay 获取指定时间所在天的开始时间
func StartOfDay(t Time) Time {
	t = t.In(Location())
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 获取指定时间所在天的结束时间
func EndOfDay(t Time) Time {
	return StartOfDay(t).AddDate(0, 0, 1).Add(-time.Nanosecond)
}

// StartOfMonth 获取指定时间所在月的开始时间
func StartOfMonth(t Time) Time {
	t = t.In(Location())
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 获取指定时间所在月的结束时间
func EndOfMonth(t Time) Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// DaysInMonth 获取某年某月天数
func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, Location()).Day()
}

// IsLeapYear 是否是闰年
func IsLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}

func firstOffset(offset []int) int {
	if len(offset) == 0 {
		return 0
	}
	return offset[0]
}

// addMonthsClamped 按自然月偏移，并把日期夹到目标月最后一天。
// 例如：2026-03-31 偏移 -1 个月 => 2026-02-28
func addMonthsClamped(t Time, months int) Time {
	t = t.In(Location())

	year, month, day := t.Date()

	q, r := divMod(int(month)-1+months, 12)
	year += q
	month = time.Month(r + 1)

	maxDay := DaysInMonth(year, month)
	if day > maxDay {
		day = maxDay
	}

	return time.Date(
		year,
		month,
		day,
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		t.Location(),
	)
}

func divMod(x, y int) (int, int) {
	q := x / y
	r := x % y

	if r < 0 {
		r += y
		q--
	}

	return q, r
}
