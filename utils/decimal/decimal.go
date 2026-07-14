// decimal 包实现任意精度的定点十进制数。
// Decimal 的零值就是 0，符合直觉。
//
// 创建新的 Decimal 最推荐使用 decimal.NewFromString，例如：
//
//	n, err := decimal.NewFromString("-123.4567")
//	n.String() // output: "-123.4567"
//
// 如果要把 Decimal 作为结构体字段使用：
//
//	type StructName struct {
//	    Number Decimal
//	}
//
// 注意：它“最多”只能表示小数点后 2^31 位的数字。
package decimal

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// DivisionPrecision 表示除法不能整除时，结果保留的小数位数。
// 用于不能整除的情况。
//
// 示例：
//
//	d1 := decimal.NewFromFloat(2).Div(decimal.NewFromFloat(3))
//	d1.String() // output: "0.6666666666666667"
//	d2 := decimal.NewFromFloat(2).Div(decimal.NewFromFloat(30000))
//	d2.String() // output: "0.0000666666666667"
//	d3 := decimal.NewFromFloat(20000).Div(decimal.NewFromFloat(3))
//	d3.String() // output: "6666.6666666666666667"
//	decimal.DivisionPrecision = 3
//	d4 := decimal.NewFromFloat(2).Div(decimal.NewFromFloat(3))
//	d4.String() // output: "0.667"
var DivisionPrecision = 16

// PowPrecisionNegativeExponent 指定负指数幂运算结果的小数部分最大精度。
// 仅在计算 Decimal 的幂且指数为负数时使用。
// 该常量适用于 Pow、PowInt32、PowBigInt；PowWithPrecision 不受它限制。
//
// 示例：
//
//	d1, err := decimal.NewFromFloat(15.2).PowInt32(-2)
//	d1.String() // output: "0.0043282548476454"
//
//	decimal.PowPrecisionNegativeExponent = 24
//	d2, err := decimal.NewFromFloat(15.2).PowInt32(-2)
//	d2.String() // output: "0.004328254847645429362881"
var PowPrecisionNegativeExponent = 16

// 如果希望 Decimal 在 JSON 中编码为数字而不是字符串，可将 MarshalJSONWithoutQuotes 设为 true。
// 这样 JSON 序列化时会输出数值，而不是字符串。
// 警告：对于位数很多的 Decimal，这样做有风险，因为很多 JSON
// 反序列化器，例如 JavaScript，会把 JSON 数字解析成 IEEE 754
// 双精度浮点数，这意味着可能会
// 在没有报错的情况下悄悄丢失精度。
var MarshalJSONWithoutQuotes = false

// ExpMaxIterations 指定使用 ExpHullAbrham 方法计算自然指数时的最大迭代次数。
// 用于限制精确计算自然指数值所需的迭代次数。
var ExpMaxIterations = 1000

// Zero 常量用于加快计算。
// 不要直接用 == 或 != 比较 Zero，请使用 decimal.Equal 或 decimal.Cmp。
var Zero = New(0, 1)

var zeroInt = big.NewInt(0)
var oneInt = big.NewInt(1)
var twoInt = big.NewInt(2)
var fourInt = big.NewInt(4)
var fiveInt = big.NewInt(5)
var tenInt = big.NewInt(10)
var twentyInt = big.NewInt(20)

var factorials = []Decimal{New(1, 0)}

// Decimal 表示一个定点十进制数，它是不可变对象。
// 数值关系：number = value * 10 ^ exp。
type Decimal struct {
	value *big.Int

	// 注意：这里必须是 int32，因为计算过程中会把它转换为 float64。
	// 如果 exp 是 64 位，可能会丢失精度。
	// 如果要表示所有可能的十进制数，
	// 可以把 exp 做成 *big.Int，但这会影响性能，而且这种数字
	// 在实际场景中并不现实。
	exp int32
}

// New returns a new fixed-point decimal, value * 10 ^ exp.
func New(value int64, exp int32) Decimal {
	return Decimal{
		value: big.NewInt(value),
		exp:   exp,
	}
}

// 示例：
//
//	NewFromInt(123).String() // output: "123"
//	NewFromInt(-10).String() // output: "-10"
func NewFromInt(value int64) Decimal {
	return Decimal{
		value: big.NewInt(value),
		exp:   0,
	}
}

// 示例：
//
//	NewFromInt(123).String() // output: "123"
//	NewFromInt(-10).String() // output: "-10"
func NewFromInt32(value int32) Decimal {
	return Decimal{
		value: big.NewInt(int64(value)),
		exp:   0,
	}
}

// 示例：
//
//	NewFromUint64(123).String() // output: "123"
func NewFromUint64(value uint64) Decimal {
	return Decimal{
		value: new(big.Int).SetUint64(value),
		exp:   0,
	}
}

// NewFromBigInt returns a new Decimal from a big.Int, value * 10 ^ exp
func NewFromBigInt(value *big.Int, exp int32) Decimal {
	return Decimal{
		value: new(big.Int).Set(value),
		exp:   exp,
	}
}

// 分母相除后，会按给定精度进行舍入。
// 示例：
//
//	d1 := NewFromBigRat(big.NewRat(0, 1), 0)    // output: "0"
//	d2 := NewFromBigRat(big.NewRat(4, 5), 1)    // output: "0.8"
//	d3 := NewFromBigRat(big.NewRat(1000, 3), 3) // output: "333.333"
//	d4 := NewFromBigRat(big.NewRat(2, 7), 4)    // output: "0.2857"
func NewFromBigRat(value *big.Rat, precision int32) Decimal {
	return Decimal{
		value: new(big.Int).Set(value.Num()),
		exp:   0,
	}.DivRound(Decimal{
		value: new(big.Int).Set(value.Denom()),
		exp:   0,
	}, precision)
}

// 末尾的 0 不会被裁剪。
//
// 示例：
//
//	d, err := NewFromString("-123.45")
//	d2, err := NewFromString(".0001")
//	d3, err := NewFromString("1.47000")
func NewFromString(value string) (Decimal, error) {
	originalInput := value
	var intString string
	var exp int64

	// 检查数字是否使用科学计数法。
	eIndex := strings.IndexAny(value, "Ee")
	if eIndex != -1 {
		expInt, err := strconv.ParseInt(value[eIndex+1:], 10, 32)
		if err != nil {
			if e, ok := err.(*strconv.NumError); ok && e.Err == strconv.ErrRange {
				return Decimal{}, fmt.Errorf("can't convert %s to decimal: fractional part too long", value)
			}
			return Decimal{}, fmt.Errorf("can't convert %s to decimal: exponent is not numeric", value)
		}
		value = value[:eIndex]
		exp = expInt
	}

	pIndex := -1
	vLen := len(value)
	for i := range vLen {
		if value[i] == '.' {
			if pIndex > -1 {
				return Decimal{}, fmt.Errorf("can't convert %s to decimal: too many .s", value)
			}
			pIndex = i
		}
	}

	if pIndex == -1 {
		// 没有小数点时，可以直接把原始字符串解析为
		// 整数。
		intString = value
	} else {
		if pIndex+1 < vLen {
			intString = value[:pIndex] + value[pIndex+1:]
		} else {
			intString = value[:pIndex]
		}
		expInt := -len(value[pIndex+1:])
		exp += int64(expInt)
	}

	var dValue *big.Int
	// strconv.ParseInt 比 new(big.Int).SetString 更快；这里是对确定不会溢出的字符串走快捷路径。
	if len(intString) <= 18 {
		parsed64, err := strconv.ParseInt(intString, 10, 64)
		if err != nil {
			return Decimal{}, fmt.Errorf("can't convert %s to decimal", value)
		}
		dValue = big.NewInt(parsed64)
	} else {
		dValue = new(big.Int)
		_, ok := dValue.SetString(intString, 10)
		if !ok {
			return Decimal{}, fmt.Errorf("can't convert %s to decimal", value)
		}
	}

	if exp < math.MinInt32 || exp > math.MaxInt32 {
		// 注意：现实中字符串几乎不可能长到这种程度。
		return Decimal{}, fmt.Errorf("can't convert %s to decimal: fractional part too long", originalInput)
	}

	return Decimal{
		value: dValue,
		exp:   int32(exp),
	}, nil
}

// 第二个参数 replRegexp 是正则表达式，用来查找需要从十进制字符串中移除的字符。
// 匹配到的所有字符都会被替换为空字符串。
//
// 示例：
//
//	r := regexp.MustCompile("[$,]")
//	d1, err := NewFromFormattedString("$5,125.99", r)
//
//	r2 := regexp.MustCompile("[_]")
//	d2, err := NewFromFormattedString("1_000_000", r2)
//
//	r3 := regexp.MustCompile("[USD\\s]")
//	d3, err := NewFromFormattedString("5000 USD", r3)
func NewFromFormattedString(value string, replRegexp *regexp.Regexp) (Decimal, error) {
	parsedValue := replRegexp.ReplaceAllString(value, "")
	d, err := NewFromString(parsedValue)
	if err != nil {
		return Decimal{}, err
	}
	return d, nil
}

// RequireFromString 根据字符串表示创建新的 Decimal。
// 如果 NewFromString 返回错误，则直接 panic。
//
// 示例：
//
//	d := RequireFromString("-123.45")
//	d2 := RequireFromString(".0001")
func RequireFromString(value string) Decimal {
	dec, err := NewFromString(value)
	if err != nil {
		panic(err)
	}
	return dec
}

// NewFromFloat converts a float64 to Decimal.
//
// 转换后的数字会包含 float 能够可靠往返转换的有效位数。
// 这些有效位可以在 float 中可靠地来回转换。
// 通常是 15 位，但某些情况下可能更多。
// 更多信息可参考该链接关于二进制浮点数十进制精度的说明（https://www.exploringbinary.com/decimal-precision-of-binary-floating-point-numbers/）。
//
// 如果想稍微提高转换速度，可以用 NewFromFloatWithExponent，并显式指定精度。
// 注意：遇到 NaN、+Inf、-Inf 会 panic。
func NewFromFloat(value float64) Decimal {
	if value == 0 {
		return New(0, 0)
	}
	return newFromFloat(value, math.Float64bits(value), &float64info)
}

// NewFromFloat32 converts a float32 to Decimal.
//
// 转换后的数字会包含 float 能够可靠往返转换的有效位数。
// 这些有效位可以在 float 中可靠地来回转换。
// 通常是 6 到 8 位，取决于输入值。
// 更多信息可参考该链接关于二进制浮点数十进制精度的说明（https://www.exploringbinary.com/decimal-precision-of-binary-floating-point-numbers/）。
//
// 如果想稍微提高转换速度，可以用 NewFromFloatWithExponent，并显式指定精度。
//
// 注意：遇到 NaN、+Inf、-Inf 会 panic。
func NewFromFloat32(value float32) Decimal {
	if value == 0 {
		return New(0, 0)
	}
	// 这里使用 XOR 是为了规避 Go issue 26285。
	a := math.Float32bits(value) ^ 0x80808080
	return newFromFloat(float64(value), uint64(a)^0x80808080, &float32info)
}

func newFromFloat(val float64, bits uint64, flt *floatInfo) Decimal {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		panic(fmt.Sprintf("Cannot create a Decimal from %v", val))
	}
	exp := int(bits>>flt.mantbits) & (1<<flt.expbits - 1)
	mant := bits & (uint64(1)<<flt.mantbits - 1)

	switch exp {
	case 0:
		// 非规格化数。
		exp++

	default:
		// 添加隐含的最高有效位。
		mant |= uint64(1) << flt.mantbits
	}
	exp += flt.bias

	var d decimal
	d.Assign(mant)
	d.Shift(exp - int(flt.mantbits))
	d.neg = bits>>(flt.expbits+flt.mantbits) != 0

	roundShortest(&d, mant, exp, flt)
	// 如果少于 19 位，可以直接用 int64 计算。
	if d.nd < 19 {
		tmp := int64(0)
		m := int64(1)
		for i := d.nd - 1; i >= 0; i-- {
			tmp += m * int64(d.d[i]-'0')
			m *= 10
		}
		if d.neg {
			tmp *= -1
		}
		return Decimal{value: big.NewInt(tmp), exp: int32(d.dp) - int32(d.nd)}
	}
	dValue := new(big.Int)
	dValue, ok := dValue.SetString(string(d.d[:d.nd]), 10)
	if ok {
		return Decimal{value: dValue, exp: int32(d.dp) - int32(d.nd)}
	}

	return NewFromFloatWithExponent(val, int32(d.dp)-int32(d.nd))
}

// NewFromFloatWithExponent converts a float64 to Decimal, with an arbitrary
// 小数位数。
//
// 示例：
//
//	NewFromFloatWithExponent(123.456, -2).String() // output: "123.46"
func NewFromFloatWithExponent(value float64, exp int32) Decimal {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		panic(fmt.Sprintf("Cannot create a Decimal from %v", value))
	}

	bits := math.Float64bits(value)
	mant := bits & (1<<52 - 1)
	exp2 := int32((bits >> 52) & (1<<11 - 1))
	sign := bits >> 63

	if exp2 == 0 {
		// 特殊值。
		if mant == 0 {
			return Decimal{}
		}
		// 次正规数/非规格化数。
		exp2++
	} else {
		// 正规数。
		mant |= 1 << 52
	}

	exp2 -= 1023 + 52

	// 对二进制值进行规格化。
	for mant&1 == 0 {
		mant = mant >> 1
		exp2++
	}

	// 当 N<0 时，要精确表示 2^N 所需的十进制小数位数最多不会超过 -N。
	if exp < 0 && exp < exp2 {
		exp = min(exp2, 0)
	}

	// 把 10^M * 2^N 表示为 5^M * 2^(M+N)。
	exp2 -= exp

	temp := big.NewInt(1)
	dMant := big.NewInt(int64(mant))
	// 应用 5^M。
	if exp > 0 {
		temp = temp.SetInt64(int64(exp))
		temp = temp.Exp(fiveInt, temp, nil)
	} else if exp < 0 {
		temp = temp.SetInt64(-int64(exp))
		temp = temp.Exp(fiveInt, temp, nil)
		dMant = dMant.Mul(dMant, temp)
		temp = temp.SetUint64(1)
	}

	// 应用 2^(M+N)。
	if exp2 > 0 {
		dMant = dMant.Lsh(dMant, uint(exp2))
	} else if exp2 < 0 {
		temp = temp.Lsh(temp, uint(-exp2))
	}

	// 进行舍入和降尺度处理。
	if exp > 0 || exp2 < 0 {
		halfDown := new(big.Int).Rsh(temp, 1)
		dMant = dMant.Add(dMant, halfDown)
		dMant = dMant.Quo(dMant, temp)
	}

	if sign == 1 {
		dMant = dMant.Neg(dMant)
	}

	return Decimal{
		value: dMant,
		exp:   exp,
	}
}

// Copy 返回一个 Decimal 副本，数值和指数相同，但 value 指针不同。
func (d Decimal) Copy() Decimal {
	d.ensureInitialized()
	return Decimal{
		value: new(big.Int).Set(d.value),
		exp:   d.exp,
	}
}

// 在给定指数大于原始指数时，精度可能降低。
// 也就是可能比原 Decimal 的初始指数更不精确。
// 注意：这里是截断，不是四舍五入。
//
// 示例：
//
//	d := New(12345, -4)
//	d2 := d.rescale(-1)
//	d3 := d2.rescale(-4)
//	println(d1)
//	println(d2)
//	println(d3)
//
// Output:
// 输出：
//
//	1.2345
//	1.2
//	1.2000
func (d Decimal) rescale(exp int32) Decimal {
	d.ensureInitialized()

	if d.exp == exp {
		return Decimal{
			new(big.Int).Set(d.value),
			d.exp,
		}
	}

	// 注意：相减前必须把 exp 转为 float64，以避免溢出。
	diff := math.Abs(float64(exp) - float64(d.exp))
	value := new(big.Int).Set(d.value)

	expScale := new(big.Int).Exp(tenInt, big.NewInt(int64(diff)), nil)
	if exp > d.exp {
		value = value.Quo(value, expScale)
	} else if exp < d.exp {
		value = value.Mul(value, expScale)
	}

	return Decimal{
		value: value,
		exp:   exp,
	}
}

// Abs 返回 Decimal 的绝对值。
func (d Decimal) Abs() Decimal {
	if !d.IsNegative() {
		return d
	}
	d.ensureInitialized()
	d2Value := new(big.Int).Abs(d.value)
	return Decimal{
		value: d2Value,
		exp:   d.exp,
	}
}

// Add 返回 d + d2。
func (d Decimal) Add(d2 Decimal) Decimal {
	rd, rd2 := RescalePair(d, d2)

	d3Value := new(big.Int).Add(rd.value, rd2.value)
	return Decimal{
		value: d3Value,
		exp:   rd.exp,
	}
}

// Sub 返回 d - d2。
func (d Decimal) Sub(d2 Decimal) Decimal {
	rd, rd2 := RescalePair(d, d2)

	d3Value := new(big.Int).Sub(rd.value, rd2.value)
	return Decimal{
		value: d3Value,
		exp:   rd.exp,
	}
}

// Neg 返回 -d。
func (d Decimal) Neg() Decimal {
	d.ensureInitialized()
	val := new(big.Int).Neg(d.value)
	return Decimal{
		value: val,
		exp:   d.exp,
	}
}

// Mul 返回 d * d2。
func (d Decimal) Mul(d2 Decimal) Decimal {
	d.ensureInitialized()
	d2.ensureInitialized()

	expInt64 := int64(d.exp) + int64(d2.exp)
	if expInt64 > math.MaxInt32 || expInt64 < math.MinInt32 {
		// 注意：与其给出错误结果，不如直接 panic，因为
		// Decimal 通常用于金额计算。
		panic(fmt.Sprintf("exponent %v overflows an int32!", expInt64))
	}

	d3Value := new(big.Int).Mul(d.value, d2.value)
	return Decimal{
		value: d3Value,
		exp:   int32(expInt64),
	}
}

// Shift 按 10 进制移动 Decimal。
// shift 为正时左移，为负时右移。
// 简单说，就是把 shift 加到 Decimal 的指数上。
// 也就是调整 Decimal 的 exp。
func (d Decimal) Shift(shift int32) Decimal {
	d.ensureInitialized()
	return Decimal{
		value: new(big.Int).Set(d.value),
		exp:   d.exp + shift,
	}
}

// Div 返回 d / d2；如果不能整除，结果会保留
// DivisionPrecision 指定的小数位数。
func (d Decimal) Div(d2 Decimal) Decimal {
	return d.DivRound(d2, int32(DivisionPrecision))
}

// QuoRem 执行带余数的除法。
// d.QuoRem(d2,precision) returns quotient q and remainder r such that
//
//	d = d2 * q + r, q an integer multiple of 10^(-precision)
//	0 <= r < abs(d2) * 10 ^(-precision) if d>=0
//	0 >= r > -abs(d2) * 10 ^(-precision) if d<0
//
// 注意：precision < 0 也是允许的输入。
func (d Decimal) QuoRem(d2 Decimal, precision int32) (Decimal, Decimal) {
	d.ensureInitialized()
	d2.ensureInitialized()
	if d2.value.Sign() == 0 {
		panic("decimal division by 0")
	}
	scale := -precision
	e := int64(d.exp) - int64(d2.exp) - int64(scale)
	if e > math.MaxInt32 || e < math.MinInt32 {
		panic("overflow in decimal QuoRem")
	}
	var aa, bb, expo big.Int
	var scalerest int32
	// d = a 10^ea
	// d2 = b 10^eb
	if e < 0 {
		aa = *d.value
		expo.SetInt64(-e)
		bb.Exp(tenInt, &expo, nil)
		bb.Mul(d2.value, &bb)
		scalerest = d.exp
		// 此时 aa = a。
		//     bb = b 10^(scale + eb - ea)
	} else {
		expo.SetInt64(e)
		aa.Exp(tenInt, &expo, nil)
		aa.Mul(d.value, &aa)
		bb = *d2.value
		scalerest = scale + d2.exp
		// 此时 aa = a ^ (ea - eb - scale)。
		//     bb = b
	}
	var q, r big.Int
	q.QuoRem(&aa, &bb, &r)
	dq := Decimal{value: &q, exp: scale}
	dr := Decimal{value: &r, exp: scalerest}
	return dq, dr
}

// DivRound 执行除法并按给定精度舍入。
// 也就是舍入到 10^(-precision) 的整数倍。
//
// 注意：precision < 0 也是允许的输入。
func (d Decimal) DivRound(d2 Decimal, precision int32) Decimal {
	// QuoRem 已经完成初始化检查。
	q, r := d.QuoRem(d2, precision)
	// 实际的舍入判断基于比较 r*10^precision 和 d2/2。
	// 这里改为比较 2*r*10^precision 和 d2。
	var rv2 big.Int
	rv2.Abs(r.value)
	rv2.Lsh(&rv2, 1)
	// 此时 rv2 = abs(r.value) * 2。
	r2 := Decimal{value: &rv2, exp: r.exp + precision}
	// 此时 r2 = 2 * r * 10 ^ precision。
	var c = r2.Cmp(d2.Abs())

	if c < 0 {
		return q
	}

	if d.value.Sign()*d2.value.Sign() < 0 {
		return q.Sub(New(1, -precision))
	}

	return q.Add(New(1, -precision))
}

// Mod 返回 d % d2。
func (d Decimal) Mod(d2 Decimal) Decimal {
	_, r := d.QuoRem(d2, 0)
	return r
}

// Pow 返回 d 的 d2 次幂。
// 当指数为负数时，返回结果最多保留 PowPrecisionNegativeExponent 位小数。
//
// Pow 对幂运算边界情况会返回 0（Decimal 零值）而不是错误；如需处理这些边界情况，请使用 PowWithPrecision。
// Pow 不处理的边界情况：
//   - 0 ** 0 => undefined value
//   - 0 ** y, where y < 0 => infinity
//   - x ** y, where x < 0 and y is non-integer decimal => imaginary value
//
// 示例：
//
//	d1 := decimal.NewFromFloat(4.0)
//	d2 := decimal.NewFromFloat(4.0)
//	res1 := d1.Pow(d2)
//	res1.String() // output: "256"
//
//	d3 := decimal.NewFromFloat(5.0)
//	d4 := decimal.NewFromFloat(5.73)
//	res2 := d3.Pow(d4)
//	res2.String() // output: "10118.08037125"
func (d Decimal) Pow(d2 Decimal) Decimal {
	baseSign := d.Sign()
	expSign := d2.Sign()

	if baseSign == 0 {
		if expSign == 0 {
			return Decimal{}
		}
		if expSign == 1 {
			return Decimal{zeroInt, 0}
		}
		if expSign == -1 {
			return Decimal{}
		}
	}

	if expSign == 0 {
		return Decimal{oneInt, 0}
	}

	// TODO：优化小数部分的提取。
	one := Decimal{oneInt, 0}
	expIntPart, expFracPart := d2.QuoRem(one, 0)

	if baseSign == -1 && !expFracPart.IsZero() {
		return Decimal{}
	}

	intPartPow, _ := d.PowBigInt(expIntPart.value)

	// 如果指数是整数，就不需要计算 d1**frac(d2)。
	if expFracPart.value.Sign() == 0 {
		return intPartPow
	}

	// TODO：优化 NumDigits，让精度调整性能更好。
	digitsBase := d.NumDigits()
	digitsExponent := d2.NumDigits()

	precision := digitsBase

	if digitsExponent > precision {
		precision += digitsExponent
	}

	precision += 6

	// 计算 x ** frac(y)，其中：
	// x ** frac(y) = exp(ln(x ** frac(y)) = exp(ln(x) * frac(y))
	fracPartPow, err := d.Abs().Ln(-d.exp + int32(precision))
	if err != nil {
		return Decimal{}
	}

	fracPartPow = fracPartPow.Mul(expFracPart)

	fracPartPow, err = fracPartPow.ExpTaylor(-d.exp + int32(precision))
	if err != nil {
		return Decimal{}
	}

	// 合并整数部分和小数部分。
	// base ** (expBase + expFrac) = base ** expBase * base ** expFrac
	res := intPartPow.Mul(fracPartPow)

	return res
}

// PowWithPrecision 返回 d 的 d2 次幂。
// precision 参数指定结果的最小精度，也就是小数位数。
// 返回的 Decimal 不会强制舍入到 precision 位小数。
//
// PowWithPrecision 在以下情况返回错误：
//   - 0 ** 0 => undefined value
//   - 0 ** y, where y < 0 => infinity
//   - x ** y, where x < 0 and y is non-integer decimal => imaginary value
//
// 示例：
//
//	d1 := decimal.NewFromFloat(4.0)
//	d2 := decimal.NewFromFloat(4.0)
//	res1, err := d1.PowWithPrecision(d2, 2)
//	res1.String() // output: "256"
//
//	d3 := decimal.NewFromFloat(5.0)
//	d4 := decimal.NewFromFloat(5.73)
//	res2, err := d3.PowWithPrecision(d4, 5)
//	res2.String() // output: "10118.080371595015625"
//
//	d5 := decimal.NewFromFloat(-3.0)
//	d6 := decimal.NewFromFloat(-6.0)
//	res3, err := d5.PowWithPrecision(d6, 10)
//	res3.String() // output: "0.0013717421"
func (d Decimal) PowWithPrecision(d2 Decimal, precision int32) (Decimal, error) {
	baseSign := d.Sign()
	expSign := d2.Sign()

	if baseSign == 0 {
		if expSign == 0 {
			return Decimal{}, fmt.Errorf("cannot represent undefined value of 0**0")
		}
		if expSign == 1 {
			return Decimal{zeroInt, 0}, nil
		}
		if expSign == -1 {
			return Decimal{}, fmt.Errorf("cannot represent infinity value of 0 ** y, where y < 0")
		}
	}

	if expSign == 0 {
		return Decimal{oneInt, 0}, nil
	}

	// TODO：优化小数部分的提取。
	one := Decimal{oneInt, 0}
	expIntPart, expFracPart := d2.QuoRem(one, 0)

	if baseSign == -1 && !expFracPart.IsZero() {
		return Decimal{}, fmt.Errorf("cannot represent imaginary value of x ** y, where x < 0 and y is non-integer decimal")
	}

	intPartPow, _ := d.powBigIntWithPrecision(expIntPart.value, precision)

	// 如果指数是整数，就不需要计算 d1**frac(d2)。
	if expFracPart.value.Sign() == 0 {
		return intPartPow, nil
	}

	// TODO：优化 NumDigits，让精度调整性能更好。
	digitsBase := d.NumDigits()
	digitsExponent := d2.NumDigits()

	if int32(digitsBase) > precision {
		precision = int32(digitsBase)
	}
	if int32(digitsExponent) > precision {
		precision += int32(digitsExponent)
	}

	// 将精度增加 10 位，用于补偿后续计算误差。
	precision += 10

	// 计算 x ** frac(y)，其中：
	// x ** frac(y) = exp(ln(x ** frac(y)) = exp(ln(x) * frac(y))
	fracPartPow, err := d.Abs().Ln(precision)
	if err != nil {
		return Decimal{}, err
	}

	fracPartPow = fracPartPow.Mul(expFracPart)

	fracPartPow, err = fracPartPow.ExpTaylor(precision)
	if err != nil {
		return Decimal{}, err
	}

	// 合并整数部分和小数部分。
	// base ** (expBase + expFrac) = base ** expBase * base ** expFrac
	res := intPartPow.Mul(fracPartPow)

	return res, nil
}

// PowInt32 返回 d 的 exp 次幂，其中 exp 是 int32。
// 仅当 d 和 exp 都为 0、结果未定义时返回错误。
//
// 当指数为负数时，返回结果最多保留 PowPrecisionNegativeExponent 位小数。
//
// 示例：
//
//	d1, err := decimal.NewFromFloat(4.0).PowInt32(4)
//	d1.String() // output: "256"
//
//	d2, err := decimal.NewFromFloat(3.13).PowInt32(5)
//	d2.String() // output: "300.4150512793"
func (d Decimal) PowInt32(exp int32) (Decimal, error) {
	if d.IsZero() && exp == 0 {
		return Decimal{}, fmt.Errorf("cannot represent undefined value of 0**0")
	}

	isExpNeg := exp < 0
	exp = abs(exp)

	n, result := d, New(1, 0)

	for exp > 0 {
		if exp%2 == 1 {
			result = result.Mul(n)
		}
		exp /= 2

		if exp > 0 {
			n = n.Mul(n)
		}
	}

	if isExpNeg {
		return New(1, 0).DivRound(result, int32(PowPrecisionNegativeExponent)), nil
	}

	return result, nil
}

// PowBigInt 返回 d 的 exp 次幂，其中 exp 是 big.Int。
// 仅当 d 和 exp 都为 0、结果未定义时返回错误。
//
// 当指数为负数时，返回结果最多保留 PowPrecisionNegativeExponent 位小数。
//
// 示例：
//
//	d1, err := decimal.NewFromFloat(3.0).PowBigInt(big.NewInt(3))
//	d1.String() // output: "27"
//
//	d2, err := decimal.NewFromFloat(629.25).PowBigInt(big.NewInt(5))
//	d2.String() // output: "98654323103449.5673828125"
func (d Decimal) PowBigInt(exp *big.Int) (Decimal, error) {
	return d.powBigIntWithPrecision(exp, int32(PowPrecisionNegativeExponent))
}

func (d Decimal) powBigIntWithPrecision(exp *big.Int, precision int32) (Decimal, error) {
	if d.IsZero() && exp.Sign() == 0 {
		return Decimal{}, fmt.Errorf("cannot represent undefined value of 0**0")
	}

	tmpExp := new(big.Int).Set(exp)
	isExpNeg := exp.Sign() < 0

	if isExpNeg {
		tmpExp.Abs(tmpExp)
	}

	n, result := d, New(1, 0)

	for tmpExp.Sign() > 0 {
		if tmpExp.Bit(0) == 1 {
			result = result.Mul(n)
		}
		tmpExp.Rsh(tmpExp, 1)

		if tmpExp.Sign() > 0 {
			n = n.Mul(n)
		}
	}

	if isExpNeg {
		return New(1, 0).DivRound(result, precision), nil
	}

	return result, nil
}

// ExpHullAbrham 使用 Hull-Abraham 算法计算 Decimal 的自然指数，即 e^d。
// overallPrecision 参数指定结果的整体精度，即整数部分加小数部分。
//
// 在低精度时 ExpHullAbrham 比 ExpTaylor 快，但在高精度时会慢很多。
//
// 示例：
//
//	NewFromFloat(26.1).ExpHullAbrham(2).String()    // output: "220000000000"
//	NewFromFloat(26.1).ExpHullAbrham(20).String()   // output: "216314672147.05767284"
func (d Decimal) ExpHullAbrham(overallPrecision uint32) (Decimal, error) {
	// 算法基于《Variable precision exponential function》。
	// 参考 T. E. Hull 与 A. Abrham 在 ACM Transactions on Mathematical Software 上的论文。
	if d.IsZero() {
		return Decimal{oneInt, 0}, nil
	}

	currentPrecision := overallPrecision

	// 如果 currentPrecision * 23 < |x|，该算法不适用。
	// 这种情况下会自动提高精度，以便精确计算结果。
	// 如果新计算出的精度超过 ExpMaxIterations，则不会修改 currentPrecision。
	f := d.Abs().InexactFloat64()
	if ncp := f / 23; ncp > float64(currentPrecision) && ncp < float64(ExpMaxIterations) {
		currentPrecision = uint32(math.Ceil(ncp))
	}

	// 如果 abs(d) 超过上溢/下溢阈值，则返回失败。
	overflowThreshold := New(23*int64(currentPrecision), 0)
	if d.Abs().Cmp(overflowThreshold) > 0 {
		return Decimal{}, fmt.Errorf("over/underflow threshold, exp(x) cannot be calculated precisely")
	}

	// 如果 abs(d) 足够小，直接返回 1；这也能避免后续上溢/下溢。
	overflowThreshold2 := New(9, -int32(currentPrecision)-1)
	if d.Abs().Cmp(overflowThreshold2) <= 0 {
		return Decimal{oneInt, d.exp}, nil
	}

	// t 是满足对应 abs(d/k) < 1 的最小非负整数。
	t := max(
		// Add d.NumDigits because the paper assumes that d.value [0.1, 1)
		d.exp+int32(d.NumDigits()), 0)

	k := New(1, t)                                     // reduction factor
	r := Decimal{new(big.Int).Set(d.value), d.exp - t} // reduced argument
	p := int32(currentPrecision) + t + 2               // precision for calculating the sum

	// 确定 n，也就是求和时需要的项数。
	// 使用第一步牛顿近似：(1.435p - 1.182) / log10(p/abs(r))。
	// 用于求解相应方程，并结合定向
	// 舍入以及 log10(p/abs(r)) 的简单有理边界。
	rf := r.Abs().InexactFloat64()
	pf := float64(p)
	nf := math.Ceil((1.453*pf - 1.182) / math.Log10(pf/rf))
	if nf > float64(ExpMaxIterations) || math.IsNaN(nf) {
		return Decimal{}, fmt.Errorf("exact value cannot be calculated in <=ExpMaxIterations iterations")
	}
	n := int64(nf)

	tmp := New(0, 0)
	sum := New(1, 0)
	one := New(1, 0)
	for i := n - 1; i > 0; i-- {
		tmp.value.SetInt64(i)
		sum = sum.Mul(r.DivRound(tmp, p))
		sum = sum.Add(one)
	}

	ki := k.IntPart()
	res := New(1, 0)
	for i := ki; i > 0; i-- {
		res = res.Mul(sum)
	}

	resNumDigits := int32(res.NumDigits())

	var roundDigits int32
	if resNumDigits > abs(res.exp) {
		roundDigits = int32(currentPrecision) - resNumDigits - res.exp
	} else {
		roundDigits = int32(currentPrecision)
	}

	res = res.Round(roundDigits)

	return res, nil
}

// ExpTaylor 使用泰勒级数展开计算 Decimal 的自然指数，即 e^d。
// precision 参数指定结果需要多精确，也就是小数位数。
// 允许使用负精度。
//
// 在高精度场景下，ExpTaylor 比 ExpHullAbrham 快得多。
//
// 示例：
//
//	d, err := NewFromFloat(26.1).ExpTaylor(2).String()
//	d.String()  // output: "216314672147.06"
//
//	NewFromFloat(26.1).ExpTaylor(20).String()
//	d.String()  // output: "216314672147.05767284062928674083"
//
//	NewFromFloat(26.1).ExpTaylor(-10).String()
//	d.String()  // output: "220000000000"
func (d Decimal) ExpTaylor(precision int32) (Decimal, error) {
	// 注意：该实现可以通过只使用 big.Int API 进一步优化。
	if d.IsZero() {
		return Decimal{oneInt, 0}.Round(precision), nil
	}

	var epsilon Decimal
	var divPrecision int32
	if precision < 0 {
		epsilon = New(1, -1)
		divPrecision = 8
	} else {
		epsilon = New(1, -precision-1)
		divPrecision = precision + 1
	}

	decAbs := d.Abs()
	pow := d.Abs()
	factorial := New(1, 0)

	result := New(1, 0)

	for i := int64(1); ; {
		step := pow.DivRound(factorial, divPrecision)
		result = result.Add(step)

		// 当当前项小于 epsilon 时停止泰勒级数迭代。
		if step.Cmp(epsilon) < 0 {
			break
		}

		pow = pow.Mul(decAbs)

		i++

		// 计算下一个阶乘值，或读取缓存值。
		if len(factorials) >= int(i) && !factorials[i-1].IsZero() {
			factorial = factorials[i-1]
		} else {
			// 为了避免竞态条件，先向切片追加一个零值，预留出
			// 新阶乘结果的位置。随后再用计算出的阶乘
			// 通过下标替换该零值。
			factorial = factorials[i-2].Mul(New(i, 0))
			factorials = append(factorials, Zero)
			factorials[i-1] = factorial
		}
	}

	if d.Sign() < 0 {
		result = New(1, 0).DivRound(result, precision+1)
	}

	result = result.Round(precision)
	return result, nil
}

// Ln 计算 d 的自然对数。
// precision 参数指定结果需要多精确，也就是小数位数。
// 允许使用负精度。
//
// 示例：
//
//	d1, err := NewFromFloat(13.3).Ln(2)
//	d1.String()  // output: "2.59"
//
//	d2, err := NewFromFloat(579.161).Ln(10)
//	d2.String()  // output: "6.3615805046"
func (d Decimal) Ln(precision int32) (Decimal, error) {
	// 算法基于《The Use of Iteration Methods for Approximating the Natural Logarithm》。
	// 作者 James F. Epperson，发表于 The American Mathematical Monthly，1989 年 11 月，第 96 卷第 9 期，第 831-835 页。
	if d.IsNegative() {
		return Decimal{}, fmt.Errorf("cannot calculate natural logarithm for negative decimals")
	}

	if d.IsZero() {
		return Decimal{}, fmt.Errorf("cannot represent natural logarithm of 0, result: -infinity")
	}

	calcPrecision := precision + 2
	z := d.Copy()

	var comp1, comp3, comp2, comp4, reduceAdjust Decimal
	comp1 = z.Sub(Decimal{oneInt, 0})
	comp3 = Decimal{oneInt, -1}

	// 对于范围 [0.9, 1.1] 内、ln(d) 接近 0 的 Decimal。
	usePowerSeries := false

	if comp1.Abs().Cmp(comp3) <= 0 {
		usePowerSeries = true
	} else {
		// 将输入 Decimal 缩放到 [0.1, 1) 范围内。
		expDelta := int32(z.NumDigits()) + z.exp
		z.exp -= expDelta

		// 输入 Decimal 被缩小了 10^expDelta 倍，因此需要额外加上
		// 以补偿前面的缩放。
		ln10 := ln10.withPrecision(calcPrecision)
		reduceAdjust = NewFromInt32(expDelta)
		reduceAdjust = reduceAdjust.Mul(ln10)

		comp1 = z.Sub(Decimal{oneInt, 0})

		if comp1.Abs().Cmp(comp3) <= 0 {
			usePowerSeries = true
		} else {
			// 使用 float 做初始估计。
			zFloat := z.InexactFloat64()
			comp1 = NewFromFloat(math.Log(zFloat))
		}
	}

	epsilon := Decimal{oneInt, -calcPrecision}

	if usePowerSeries {

		// 幂级数，参考该链接的对数幂级数说明。
		// 计算公式第 n 项：ln(z+1) = 2 * sum[1/(2n+1) * (z/(z+2))^(2n+1)]。
		// 直到当前项和下一项的差值小于 epsilon。
		// 对于接近 1.0 的 Decimal，收敛很快。

		// z + 2
		comp2 = comp1.Add(Decimal{twoInt, 0})
		// z / (z + 2)
		comp3 = comp1.DivRound(comp2, calcPrecision)
		// 2 * (z / (z + 2))
		comp1 = comp3.Add(comp3)
		comp2 = comp1.Copy()

		for n := 1; ; n++ {
			// 2 * (z / (z+2))^(2n+1)
			comp2 = comp2.Mul(comp3).Mul(comp3)

			// 1 / (2n+1) * 2 * (z / (z+2))^(2n+1)
			comp4 = NewFromInt(int64(2*n + 1))
			comp4 = comp2.DivRound(comp4, calcPrecision)

			// comp1 = 2 sum [ 1 / (2n+1) * (z / (z+2))^(2n+1) ]
			comp1 = comp1.Add(comp4)

			if comp4.Abs().Cmp(epsilon) <= 0 {
				break
			}
		}
	} else {
		// Halley 迭代法。
		// 计算公式第 n 项：a_(n+1) = a_n - 2 * (exp(a_n) - z) / (exp(a_n) + z)。
		// 直到当前项和下一项的差值小于 epsilon。
		var prevStep Decimal
		maxIters := calcPrecision*2 + 10

		for range maxIters {
			// exp(a_n)
			comp3, _ = comp1.ExpTaylor(calcPrecision)
			// exp(a_n) - z
			comp2 = comp3.Sub(z)
			// 2 * (exp(a_n) - z)
			comp2 = comp2.Add(comp2)
			// exp(a_n) + z
			comp4 = comp3.Add(z)
			// 2 * (exp(a_n) - z) / (exp(a_n) + z)
			comp3 = comp2.DivRound(comp4, calcPrecision)
			// comp1 = a_(n+1) = a_n - 2 * (exp(a_n) - z) / (exp(a_n) + z)
			comp1 = comp1.Sub(comp3)

			if prevStep.Add(comp3).IsZero() {
				// 如果迭代步发生振荡，应提前返回，避免无限循环。
				// 注意：这种情况应该很少见，不需要返回错误。
				break
			}

			if comp3.Abs().Cmp(epsilon) <= 0 {
				break
			}

			prevStep = comp3
		}
	}

	comp1 = comp1.Add(reduceAdjust)

	return comp1.Round(precision), nil
}

// NumDigits 返回 Decimal 系数部分的数字位数，也就是 d.Value 的位数。
func (d Decimal) NumDigits() int {
	if d.value == nil {
		return 1
	}

	if d.value.IsInt64() {
		i64 := d.value.Int64()
		// restrict fast path to integers with exact conversion to float64
		if i64 <= (1<<53) && i64 >= -(1<<53) {
			if i64 == 0 {
				return 1
			}
			return int(math.Log10(math.Abs(float64(i64)))) + 1
		}
	}

	estimatedNumDigits := int(float64(d.value.BitLen()) / math.Log2(10))

	// estimatedNumDigits（log10 估算）可能误差 1 位，需要校验。
	digitsBigInt := big.NewInt(int64(estimatedNumDigits))
	errorCorrectionUnit := digitsBigInt.Exp(tenInt, digitsBigInt, nil)

	if d.value.CmpAbs(errorCorrectionUnit) >= 0 {
		return estimatedNumDigits + 1
	}

	return estimatedNumDigits
}

// 当 Decimal 可以表示为整数时，IsInteger 返回 true；否则返回 false。
func (d Decimal) IsInteger() bool {

	// 最常见的情况：所有指数大于等于 0 的 Decimal 都可以表示为整数。
	if d.exp >= 0 {
		return true
	}
	// 当指数为负数时，需要检查小数点后的每一位。
	// 如果小数部分全是 0，就可以确定该 Decimal 能表示为整数。
	var r big.Int
	q := new(big.Int).Set(d.value)
	for z := abs(d.exp); z > 0; z-- {
		q.QuoRem(q, tenInt, &r)
		if r.Cmp(zeroInt) != 0 {
			return false
		}
	}
	return true
}

// abs 计算 int32 的绝对值，用于计算 Decimal 指数的绝对值。
func abs(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}

// Cmp 比较 d 和 d2 表示的数值，并返回：
//
//	-1 if d <  d2
//	 0 if d == d2
//	+1 if d >  d2
func (d Decimal) Cmp(d2 Decimal) int {
	d.ensureInitialized()
	d2.ensureInitialized()

	if d.exp == d2.exp {
		return d.value.Cmp(d2.value)
	}

	rd, rd2 := RescalePair(d, d2)

	return rd.value.Cmp(rd2.value)
}

// Compare 比较 d 和 d2 表示的数值，并返回：
//
//	-1 if d <  d2
//	 0 if d == d2
//	+1 if d >  d2
func (d Decimal) Compare(d2 Decimal) int {
	return d.Cmp(d2)
}

// Equal 返回 d 和 d2 表示的数值是否相等。
func (d Decimal) Equal(d2 Decimal) bool {
	return d.Cmp(d2) == 0
}

// 已废弃：Equals 已废弃，请改用 Equal 方法。
func (d Decimal) Equals(d2 Decimal) bool {
	return d.Equal(d2)
}

// 当 d 大于 d2 时，GreaterThan（GT）返回 true。
func (d Decimal) GreaterThan(d2 Decimal) bool {
	return d.Cmp(d2) == 1
}

// 当 d 大于或等于 d2 时，GreaterThanOrEqual（GTE）返回 true。
func (d Decimal) GreaterThanOrEqual(d2 Decimal) bool {
	cmp := d.Cmp(d2)
	return cmp == 1 || cmp == 0
}

// 当 d 小于 d2 时，LessThan（LT）返回 true。
func (d Decimal) LessThan(d2 Decimal) bool {
	return d.Cmp(d2) == -1
}

// 当 d 小于或等于 d2 时，LessThanOrEqual（LTE）返回 true。
func (d Decimal) LessThanOrEqual(d2 Decimal) bool {
	cmp := d.Cmp(d2)
	return cmp == -1 || cmp == 0
}

// Sign 返回：
//
//	-1 if d <  0
//	 0 if d == 0
//	+1 if d >  0
func (d Decimal) Sign() int {
	if d.value == nil {
		return 0
	}
	return d.value.Sign()
}

// IsPositive 返回：
//
//	true if d > 0
//	false if d == 0
//	false if d < 0
func (d Decimal) IsPositive() bool {
	return d.Sign() == 1
}

// IsNegative 返回：
//
//	true if d < 0
//	false if d == 0
//	false if d > 0
func (d Decimal) IsNegative() bool {
	return d.Sign() == -1
}

// IsZero 返回：
//
//	true if d == 0
//	false if d > 0
//	false if d < 0
func (d Decimal) IsZero() bool {
	return d.Sign() == 0
}

// Exponent 返回 Decimal 的指数，也就是 scale 部分。
func (d Decimal) Exponent() int32 {
	return d.exp
}

// Coefficient 返回 Decimal 的系数部分，该系数按 10^Exponent() 缩放。
func (d Decimal) Coefficient() *big.Int {
	d.ensureInitialized()
	// 这里复制系数，避免修改返回值时影响原 Decimal。
	return new(big.Int).Set(d.value)
}

// CoefficientInt64 以 int64 返回 Decimal 的系数，该系数按 10^Exponent() 缩放。
// 如果系数不能用 int64 表示，结果未定义。
func (d Decimal) CoefficientInt64() int64 {
	d.ensureInitialized()
	return d.value.Int64()
}

// IntPart 返回 Decimal 的整数部分。
func (d Decimal) IntPart() int64 {
	scaledD := d.rescale(0)
	return scaledD.value.Int64()
}

// BigInt 以 BigInt 返回 Decimal 的整数部分。
func (d Decimal) BigInt() *big.Int {
	scaledD := d.rescale(0)
	return scaledD.value
}

// BigFloat 将 Decimal 转换为 BigFloat。
// 注意：把 Decimal 转为 BigFloat 可能会丢失精度。
func (d Decimal) BigFloat() *big.Float {
	f := &big.Float{}
	f.SetString(d.String())
	return f
}

// Rat 返回 Decimal 的有理数表示。
func (d Decimal) Rat() *big.Rat {
	d.ensureInitialized()
	if d.exp <= 0 {
		// 注意：必须在类型转换后再取负，以避免 int32 溢出。
		denom := new(big.Int).Exp(tenInt, big.NewInt(-int64(d.exp)), nil)
		return new(big.Rat).SetFrac(d.value, denom)
	}

	mul := new(big.Int).Exp(tenInt, big.NewInt(int64(d.exp)), nil)
	num := new(big.Int).Mul(d.value, mul)
	return new(big.Rat).SetFrac(num, oneInt)
}

// Float64 返回最接近 d 的 float64 值，以及一个 bool 表示
// 该 float64 是否能精确表示 d。
// 更多细节请查看 big.Rat.Float64 的文档。
func (d Decimal) Float64() (f float64, exact bool) {
	return d.Rat().Float64()
}

// InexactFloat64 返回最接近 d 的 float64 值。
// 它不会说明返回值是否能精确表示 d。
func (d Decimal) InexactFloat64() float64 {
	f, _ := d.Float64()
	return f
}

// String 返回 Decimal 的字符串表示。
// 采用定点格式。
//
// 示例：
//
//	d := New(-12345, -3)
//	println(d.String())
//
// Output:
// 输出：
//
//	-12.345
func (d Decimal) String() string {
	return d.string(true)
}

// StringFixed 返回舍入后的定点字符串，保留 places 位
// 小数。
//
// 示例：
//
//	NewFromFloat(0).StringFixed(2) // output: "0.00"
//	NewFromFloat(0).StringFixed(0) // output: "0"
//	NewFromFloat(5.45).StringFixed(0) // output: "5"
//	NewFromFloat(5.45).StringFixed(1) // output: "5.5"
//	NewFromFloat(5.45).StringFixed(2) // output: "5.45"
//	NewFromFloat(5.45).StringFixed(3) // output: "5.450"
//	NewFromFloat(545).StringFixed(-1) // output: "550"
func (d Decimal) StringFixed(places int32) string {
	rounded := d.Round(places)
	return rounded.string(false)
}

// StringFixedBank 返回银行家舍入后的定点字符串，保留 places 位
// 小数。
//
// 示例：
//
//	NewFromFloat(0).StringFixedBank(2) // output: "0.00"
//	NewFromFloat(0).StringFixedBank(0) // output: "0"
//	NewFromFloat(5.45).StringFixedBank(0) // output: "5"
//	NewFromFloat(5.45).StringFixedBank(1) // output: "5.4"
//	NewFromFloat(5.45).StringFixedBank(2) // output: "5.45"
//	NewFromFloat(5.45).StringFixedBank(3) // output: "5.450"
//	NewFromFloat(545).StringFixedBank(-1) // output: "540"
func (d Decimal) StringFixedBank(places int32) string {
	rounded := d.RoundBank(places)
	return rounded.string(false)
}

// StringFixedCash 返回瑞典式/现金舍入后的定点字符串。
// 更多细节请查看 RoundCash 函数文档。
func (d Decimal) StringFixedCash(interval uint8) string {
	rounded := d.RoundCash(interval)
	return rounded.string(false)
}

// Round 将 Decimal 舍入到 places 位小数。
// 如果 places < 0，则把整数部分舍入到最接近的 10^(-places)。
//
// 示例：
//
//	NewFromFloat(5.45).Round(1).String() // output: "5.5"
//	NewFromFloat(545).Round(-1).String() // output: "550"
func (d Decimal) Round(places int32) Decimal {
	if d.exp == -places {
		return d
	}
	// 先截断到 places + 1 位。
	ret := d.rescale(-places - 1)

	// 加上 sign(d) * 0.5。
	if ret.value.Sign() < 0 {
		ret.value.Sub(ret.value, fiveInt)
	} else {
		ret.value.Add(ret.value, fiveInt)
	}

	// 正数向下取整，负数向上取整。
	_, m := ret.value.DivMod(ret.value, tenInt, new(big.Int))
	ret.exp++
	if ret.value.Sign() < 0 && m.Cmp(zeroInt) != 0 {
		ret.value.Add(ret.value, oneInt)
	}

	return ret
}

// RoundCeil 将 Decimal 向正无穷方向舍入。
//
// 示例：
//
//	NewFromFloat(545).RoundCeil(-2).String()   // output: "600"
//	NewFromFloat(500).RoundCeil(-2).String()   // output: "500"
//	NewFromFloat(1.1001).RoundCeil(2).String() // output: "1.11"
//	NewFromFloat(-1.454).RoundCeil(1).String() // output: "-1.4"
func (d Decimal) RoundCeil(places int32) Decimal {
	if d.exp >= -places {
		return d
	}

	rescaled := d.rescale(-places)
	if d.Equal(rescaled) {
		return d
	}

	if d.value.Sign() > 0 {
		rescaled.value.Add(rescaled.value, oneInt)
	}

	return rescaled
}

// RoundFloor 将 Decimal 向负无穷方向舍入。
//
// 示例：
//
//	NewFromFloat(545).RoundFloor(-2).String()   // output: "500"
//	NewFromFloat(-500).RoundFloor(-2).String()   // output: "-500"
//	NewFromFloat(1.1001).RoundFloor(2).String() // output: "1.1"
//	NewFromFloat(-1.454).RoundFloor(1).String() // output: "-1.5"
func (d Decimal) RoundFloor(places int32) Decimal {
	if d.exp >= -places {
		return d
	}

	rescaled := d.rescale(-places)
	if d.Equal(rescaled) {
		return d
	}

	if d.value.Sign() < 0 {
		rescaled.value.Sub(rescaled.value, oneInt)
	}

	return rescaled
}

// RoundUp 将 Decimal 远离 0 方向舍入。
//
// 示例：
//
//	NewFromFloat(545).RoundUp(-2).String()   // output: "600"
//	NewFromFloat(500).RoundUp(-2).String()   // output: "500"
//	NewFromFloat(1.1001).RoundUp(2).String() // output: "1.11"
//	NewFromFloat(-1.454).RoundUp(1).String() // output: "-1.5"
func (d Decimal) RoundUp(places int32) Decimal {
	if d.exp >= -places {
		return d
	}

	rescaled := d.rescale(-places)
	if d.Equal(rescaled) {
		return d
	}

	if d.value.Sign() > 0 {
		rescaled.value.Add(rescaled.value, oneInt)
	} else if d.value.Sign() < 0 {
		rescaled.value.Sub(rescaled.value, oneInt)
	}

	return rescaled
}

// RoundDown 将 Decimal 向 0 方向舍入。
//
// 示例：
//
//	NewFromFloat(545).RoundDown(-2).String()   // output: "500"
//	NewFromFloat(-500).RoundDown(-2).String()   // output: "-500"
//	NewFromFloat(1.1001).RoundDown(2).String() // output: "1.1"
//	NewFromFloat(-1.454).RoundDown(1).String() // output: "-1.4"
func (d Decimal) RoundDown(places int32) Decimal {
	if d.exp >= -places {
		return d
	}

	rescaled := d.rescale(-places)
	if d.Equal(rescaled) {
		return d
	}
	return rescaled
}

// RoundBank 将 Decimal 按银行家舍入到 places 位小数。
// 如果待舍入的最后一位到相邻两个整数距离相等，
// 则取偶数作为舍入结果。
//
// 如果 places < 0，则把整数部分舍入到最接近的 10^(-places)。
//
// 示例：
//
//	NewFromFloat(5.45).RoundBank(1).String() // output: "5.4"
//	NewFromFloat(545).RoundBank(-1).String() // output: "540"
//	NewFromFloat(5.46).RoundBank(1).String() // output: "5.5"
//	NewFromFloat(546).RoundBank(-1).String() // output: "550"
//	NewFromFloat(5.55).RoundBank(1).String() // output: "5.6"
//	NewFromFloat(555).RoundBank(-1).String() // output: "560"
func (d Decimal) RoundBank(places int32) Decimal {

	round := d.Round(places)
	remainder := d.Sub(round).Abs()

	half := New(5, -places-1)
	if remainder.Cmp(half) == 0 && round.value.Bit(0) != 0 {
		if round.value.Sign() < 0 {
			round.value.Add(round.value, oneInt)
		} else {
			round.value.Sub(round.value, oneInt)
		}
	}

	return round
}

// RoundCash，也叫现金/分/öre 舍入，会把 Decimal 舍入到指定
// 间隔。现金交易的应付金额会被舍入到最接近的
// 可用最小货币单位的倍数。支持以下间隔：
// 5、10、25、50、100；其他值会 panic。
//
//	  5:   5 cent rounding 3.43 => 3.45
//	 10:  10 cent rounding 3.45 => 3.50 (5 gets rounded up)
//	 25:  25 cent rounding 3.41 => 3.50
//	 50:  50 cent rounding 3.75 => 4.00
//	100: 100 cent rounding 3.50 => 4.00
//
// 更多细节可参考现金舍入说明（https://en.wikipedia.org/wiki/Cash_rounding）。
func (d Decimal) RoundCash(interval uint8) Decimal {
	var iVal *big.Int
	switch interval {
	case 5:
		iVal = twentyInt
	case 10:
		iVal = tenInt
	case 25:
		iVal = fourInt
	case 50:
		iVal = twoInt
	case 100:
		iVal = oneInt
	default:
		panic(fmt.Sprintf("Decimal does not support this Cash rounding interval `%d`. Supported: 5, 10, 25, 50, 100", interval))
	}
	dVal := Decimal{
		value: iVal,
	}

	// TODO：优化这些计算，减少较高的内存分配（约 29 次分配）。
	return d.Mul(dVal).Round(0).Div(dVal).Truncate(2)
}

// Floor 返回小于或等于 d 的最接近整数。
func (d Decimal) Floor() Decimal {
	d.ensureInitialized()

	if d.exp >= 0 {
		return d
	}

	exp := big.NewInt(10)

	// 注意：必须在类型转换后再取负，以避免 int32 溢出。
	exp.Exp(exp, big.NewInt(-int64(d.exp)), nil)

	z := new(big.Int).Div(d.value, exp)
	return Decimal{value: z, exp: 0}
}

// Ceil 返回大于或等于 d 的最接近整数。
func (d Decimal) Ceil() Decimal {
	d.ensureInitialized()

	if d.exp >= 0 {
		return d
	}

	exp := big.NewInt(10)

	// 注意：必须在类型转换后再取负，以避免 int32 溢出。
	exp.Exp(exp, big.NewInt(-int64(d.exp)), nil)

	z, m := new(big.Int).DivMod(d.value, exp, new(big.Int))
	if m.Cmp(zeroInt) != 0 {
		z.Add(z, oneInt)
	}
	return Decimal{value: z, exp: 0}
}

// Truncate 截断数字，不进行舍入。
//
// 注意：precision 表示不会被截断的最后一位，必须 >= 0。
//
// 示例：
//
//	decimal.NewFromString("123.456").Truncate(2).String() // "123.45"
func (d Decimal) Truncate(precision int32) Decimal {
	d.ensureInitialized()
	if precision >= 0 && -precision > d.exp {
		return d.rescale(-precision)
	}
	return d
}

// UnmarshalJSON 实现 json.Unmarshaler 接口。
func (d *Decimal) UnmarshalJSON(decimalBytes []byte) error {
	if string(decimalBytes) == "null" {
		return nil
	}

	str, err := unquoteIfQuoted(decimalBytes)
	if err != nil {
		return fmt.Errorf("error decoding string '%s': %s", decimalBytes, err)
	}

	decimal, err := NewFromString(str)
	*d = decimal
	if err != nil {
		return fmt.Errorf("error decoding string '%s': %s", str, err)
	}
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口。
func (d Decimal) MarshalJSON() ([]byte, error) {
	var str string
	if MarshalJSONWithoutQuotes {
		str = d.String()
	} else {
		str = "\"" + d.String() + "\""
	}
	return []byte(str), nil
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler 接口。由于文本编码已经使用字符串表示，
// 该方法会把该字符串保存为 []byte。
func (d *Decimal) UnmarshalBinary(data []byte) error {
	// 确认至少有 4 个字节用于存放指数。GOB 编码值
	// 可能为空。
	if len(data) < 4 {
		return fmt.Errorf("error decoding binary %v: expected at least 4 bytes, got %d", data, len(data))
	}

	// 提取指数。
	d.exp = int32(binary.BigEndian.Uint32(data[:4]))

	// 提取值。
	d.value = new(big.Int)
	if err := d.value.GobDecode(data[4:]); err != nil {
		return fmt.Errorf("error decoding binary %v: %s", data, err)
	}

	return nil
}

// MarshalBinary 实现 encoding.BinaryMarshaler 接口。
func (d Decimal) MarshalBinary() (data []byte, err error) {
	// exp 会写在前面，但为了知道输出大小，先编码 value。
	var valueData []byte
	if valueData, err = d.value.GobEncode(); err != nil {
		return nil, err
	}

	// 由于指数大小固定，所以将指数写到前面。
	expData := make([]byte, 4, len(valueData)+4)
	binary.BigEndian.PutUint32(expData, uint32(d.exp))

	// 返回字节数组。
	return append(expData, valueData...), nil
}

// Scan 实现 sql.Scanner 接口，用于数据库反序列化。
func (d *Decimal) Scan(value any) error {
	// 先尝试判断数据库中是否以 Numeric 类型存储数据。
	switch v := value.(type) {

	case float32:
		*d = NewFromFloat(float64(v))
		return nil

	case float64:
		// sqlite3 中的 numeric 会以 float64 传过来。
		*d = NewFromFloat(v)
		return nil

	case int64:

		// 至少在 sqlite3 中，当数据库里的值为 0 时，数据会以
		// int64 而不是 float64 传给我们。
		*d = New(v, 0)
		return nil

	case uint64:
		// 而 ClickHouse 可能会把数据库中的 0 作为 uint64 传过来。
		*d = NewFromUint64(v)
		return nil

	default:
		// 默认情况下尝试把存储值按字符串解析。
		str, err := unquoteIfQuoted(v)
		if err != nil {
			return err
		}
		*d, err = NewFromString(str)
		return err
	}
}

// Value 实现 driver.Valuer 接口，用于数据库序列化。
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// UnmarshalText 实现 encoding.TextUnmarshaler 接口，用于 XML
// 反序列化。
func (d *Decimal) UnmarshalText(text []byte) error {
	str := string(text)

	dec, err := NewFromString(str)
	*d = dec
	if err != nil {
		return fmt.Errorf("error decoding string '%s': %s", str, err)
	}

	return nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，用于 XML
// 序列化。
func (d Decimal) MarshalText() (text []byte, err error) {
	return []byte(d.String()), nil
}

// GobEncode 实现 gob.GobEncoder 接口，用于 gob 序列化。
func (d Decimal) GobEncode() ([]byte, error) {
	return d.MarshalBinary()
}

// GobDecode 实现 gob.GobDecoder 接口，用于 gob 反序列化。
func (d *Decimal) GobDecode(data []byte) error {
	return d.UnmarshalBinary(data)
}

func (d Decimal) string(trimTrailingZeros bool) string {
	if d.exp >= 0 {
		return d.rescale(0).value.String()
	}

	abs := new(big.Int).Abs(d.value)
	str := abs.String()

	var intPart, fractionalPart string

	// 注意：如果 d.exp == INT_MIN，这里转换为 int 会导致 bug。
	// 并且运行在 32 位机器上。这个极端边界情况不会修复。
	dExpInt := int(d.exp)
	if len(str) > -dExpInt {
		intPart = str[:len(str)+dExpInt]
		fractionalPart = str[len(str)+dExpInt:]
	} else {
		intPart = "0"

		num0s := -dExpInt - len(str)
		fractionalPart = strings.Repeat("0", num0s) + str
	}

	if trimTrailingZeros {
		i := len(fractionalPart) - 1
		for ; i >= 0; i-- {
			if fractionalPart[i] != '0' {
				break
			}
		}
		fractionalPart = fractionalPart[:i+1]
	}

	number := intPart
	if len(fractionalPart) > 0 {
		number += "." + fractionalPart
	}

	if d.value.Sign() < 0 {
		return "-" + number
	}

	return number
}

func (d *Decimal) ensureInitialized() {
	if d.value == nil {
		d.value = new(big.Int)
	}
}

// Min 返回传入参数中最小的 Decimal。
// 如果要用数组调用该函数，必须这样写：
//
//	Min(arr[0], arr[1:]...)
//
// 这样可以避免误调用没有参数的 Min。
func Min(first Decimal, rest ...Decimal) Decimal {
	ans := first
	for _, item := range rest {
		if item.Cmp(ans) < 0 {
			ans = item
		}
	}
	return ans
}

// Max 返回传入参数中最大的 Decimal。

// 如果要用数组调用该函数，必须这样写：
//
//	Max(arr[0], arr[1:]...)
//
// 这样可以避免误调用没有参数的 Max。
func Max(first Decimal, rest ...Decimal) Decimal {
	ans := first
	for _, item := range rest {
		if item.Cmp(ans) > 0 {
			ans = item
		}
	}
	return ans
}

// Sum 返回 first 和 rest 中所有 Decimal 的总和。
func Sum(first Decimal, rest ...Decimal) Decimal {
	total := first
	for _, item := range rest {
		total = total.Add(item)
	}

	return total
}

// Avg 返回 first 和 rest 中所有 Decimal 的平均值。
func Avg(first Decimal, rest ...Decimal) Decimal {
	count := New(int64(len(rest)+1), 0)
	sum := Sum(first, rest...)
	return sum.Div(count)
}

// RescalePair 将两个 Decimal 重缩放到共同指数，也就是两者中较小的 exp。
func RescalePair(d1 Decimal, d2 Decimal) (Decimal, Decimal) {
	d1.ensureInitialized()
	d2.ensureInitialized()

	if d1.exp < d2.exp {
		return d1, d2.rescale(d1.exp)
	} else if d1.exp > d2.exp {
		return d1.rescale(d2.exp), d2
	}

	return d1, d2
}

func unquoteIfQuoted(value any) (string, error) {
	var bytes []byte

	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return "", fmt.Errorf("could not convert value '%+v' to byte array of type '%T'", value, value)
	}

	// 如果金额带引号，则去掉引号。
	if len(bytes) > 2 && bytes[0] == '"' && bytes[len(bytes)-1] == '"' {
		bytes = bytes[1 : len(bytes)-1]
	}
	return string(bytes), nil
}

// NullDecimal 表示可为空的 Decimal，兼容
// 从数据库扫描 NULL 值。
type NullDecimal struct {
	Decimal Decimal
	Valid   bool
}

func NewNullDecimal(d Decimal) NullDecimal {
	return NullDecimal{
		Decimal: d,
		Valid:   true,
	}
}

// Scan 实现 sql.Scanner 接口，用于数据库反序列化。
func (d *NullDecimal) Scan(value any) error {
	if value == nil {
		d.Valid = false
		return nil
	}
	d.Valid = true
	return d.Decimal.Scan(value)
}

// Value 实现 driver.Valuer 接口，用于数据库序列化。
func (d NullDecimal) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.Decimal.Value()
}

// UnmarshalJSON 实现 json.Unmarshaler 接口。
func (d *NullDecimal) UnmarshalJSON(decimalBytes []byte) error {
	if string(decimalBytes) == "null" {
		d.Valid = false
		return nil
	}
	d.Valid = true
	return d.Decimal.UnmarshalJSON(decimalBytes)
}

// MarshalJSON 实现 json.Marshaler 接口。
func (d NullDecimal) MarshalJSON() ([]byte, error) {
	if !d.Valid {
		return []byte("null"), nil
	}
	return d.Decimal.MarshalJSON()
}

// UnmarshalText 实现 encoding.TextUnmarshaler 接口，用于 XML
func (d *NullDecimal) UnmarshalText(text []byte) error {
	str := string(text)

	// 检查空 XML 或没有正文的 XML，例如 <tag></tag>。
	if str == "" {
		d.Valid = false
		return nil
	}
	if err := d.Decimal.UnmarshalText(text); err != nil {
		d.Valid = false
		return err
	}
	d.Valid = true
	return nil
}

// MarshalText 实现 encoding.TextMarshaler 接口，用于 XML
// 序列化。
func (d NullDecimal) MarshalText() (text []byte, err error) {
	if !d.Valid {
		return []byte{}, nil
	}
	return d.Decimal.MarshalText()
}

// 三角函数。
// Atan 返回 x 的反正切值，单位为弧度。
func (d Decimal) Atan() Decimal {
	if d.Equal(NewFromFloat(0.0)) {
		return d
	}
	if d.GreaterThan(NewFromFloat(0.0)) {
		return d.satan()
	}
	return d.Neg().satan().Neg()
}

func (d Decimal) xatan() Decimal {
	P0 := NewFromFloat(-8.750608600031904122785e-01)
	P1 := NewFromFloat(-1.615753718733365076637e+01)
	P2 := NewFromFloat(-7.500855792314704667340e+01)
	P3 := NewFromFloat(-1.228866684490136173410e+02)
	P4 := NewFromFloat(-6.485021904942025371773e+01)
	Q0 := NewFromFloat(2.485846490142306297962e+01)
	Q1 := NewFromFloat(1.650270098316988542046e+02)
	Q2 := NewFromFloat(4.328810604912902668951e+02)
	Q3 := NewFromFloat(4.853903996359136964868e+02)
	Q4 := NewFromFloat(1.945506571482613964425e+02)
	z := d.Mul(d)
	b1 := P0.Mul(z).Add(P1).Mul(z).Add(P2).Mul(z).Add(P3).Mul(z).Add(P4).Mul(z)
	b2 := z.Add(Q0).Mul(z).Add(Q1).Mul(z).Add(Q2).Mul(z).Add(Q3).Mul(z).Add(Q4)
	z = b1.Div(b2)
	z = d.Mul(z).Add(d)
	return z
}

// satan 会缩小参数范围；已知参数为正数。
// 将参数缩小到 [0, 0.66] 范围后调用 xatan。
func (d Decimal) satan() Decimal {
	Morebits := NewFromFloat(6.123233995736765886130e-17) // pi/2 = PIO2 + Morebits
	Tan3pio8 := NewFromFloat(2.41421356237309504880)      // tan(3*pi/8)
	pi := NewFromFloat(3.14159265358979323846264338327950288419716939937510582097494459)

	if d.LessThanOrEqual(NewFromFloat(0.66)) {
		return d.xatan()
	}
	if d.GreaterThan(Tan3pio8) {
		return pi.Div(NewFromFloat(2.0)).Sub(NewFromFloat(1.0).Div(d).xatan()).Add(Morebits)
	}
	return pi.Div(NewFromFloat(4.0)).Add((d.Sub(NewFromFloat(1.0)).Div(d.Add(NewFromFloat(1.0)))).xatan()).Add(NewFromFloat(0.5).Mul(Morebits))
}

// sin 系数。
var _sin = [...]Decimal{
	NewFromFloat(1.58962301576546568060e-10), // 0x3de5d8fd1fd19ccd
	NewFromFloat(-2.50507477628578072866e-8), // 0xbe5ae5e5a9291f5d
	NewFromFloat(2.75573136213857245213e-6),  // 0x3ec71de3567d48a1
	NewFromFloat(-1.98412698295895385996e-4), // 0xbf2a01a019bfdf03
	NewFromFloat(8.33333333332211858878e-3),  // 0x3f8111111110f7d0
	NewFromFloat(-1.66666666666666307295e-1), // 0xbfc5555555555548
}

// Sin 返回弧度参数 x 的正弦值。
func (d Decimal) Sin() Decimal {
	PI4A := NewFromFloat(7.85398125648498535156e-1)                             // 0x3fe921fb40000000, Pi/4 split into three parts
	PI4B := NewFromFloat(3.77489470793079817668e-8)                             // 0x3e64442d00000000,
	PI4C := NewFromFloat(2.69515142907905952645e-15)                            // 0x3ce8469898cc5170,
	M4PI := NewFromFloat(1.273239544735162542821171882678754627704620361328125) // 4/pi

	if d.Equal(NewFromFloat(0.0)) {
		return d
	}
	// 将参数转为正数，但保留符号。
	sign := false
	if d.LessThan(NewFromFloat(0.0)) {
		d = d.Neg()
		sign = true
	}

	j := d.Mul(M4PI).IntPart()    // integer part of x/(Pi/4), as integer for tests on the phase angle
	y := NewFromFloat(float64(j)) // integer part of x/(Pi/4), as float

	// 将零值映射到原点。
	if j&1 == 1 {
		j++
		y = y.Add(NewFromFloat(1.0))
	}
	j &= 7 // octant modulo 2Pi radians (360 degrees)
	// 沿 x 轴反射。
	if j > 3 {
		sign = !sign
		j -= 4
	}
	z := d.Sub(y.Mul(PI4A)).Sub(y.Mul(PI4B)).Sub(y.Mul(PI4C)) // Extended precision modular arithmetic
	zz := z.Mul(z)

	if j == 1 || j == 2 {
		w := zz.Mul(zz).Mul(_cos[0].Mul(zz).Add(_cos[1]).Mul(zz).Add(_cos[2]).Mul(zz).Add(_cos[3]).Mul(zz).Add(_cos[4]).Mul(zz).Add(_cos[5]))
		y = NewFromFloat(1.0).Sub(NewFromFloat(0.5).Mul(zz)).Add(w)
	} else {
		y = z.Add(z.Mul(zz).Mul(_sin[0].Mul(zz).Add(_sin[1]).Mul(zz).Add(_sin[2]).Mul(zz).Add(_sin[3]).Mul(zz).Add(_sin[4]).Mul(zz).Add(_sin[5])))
	}
	if sign {
		y = y.Neg()
	}
	return y
}

// cos 系数。
var _cos = [...]Decimal{
	NewFromFloat(-1.13585365213876817300e-11), // 0xbda8fa49a0861a9b
	NewFromFloat(2.08757008419747316778e-9),   // 0x3e21ee9d7b4e3f05
	NewFromFloat(-2.75573141792967388112e-7),  // 0xbe927e4f7eac4bc6
	NewFromFloat(2.48015872888517045348e-5),   // 0x3efa01a019c844f5
	NewFromFloat(-1.38888888888730564116e-3),  // 0xbf56c16c16c14f91
	NewFromFloat(4.16666666666665929218e-2),   // 0x3fa555555555554b
}

// Cos 返回弧度参数 x 的余弦值。
func (d Decimal) Cos() Decimal {

	PI4A := NewFromFloat(7.85398125648498535156e-1)                             // 0x3fe921fb40000000, Pi/4 split into three parts
	PI4B := NewFromFloat(3.77489470793079817668e-8)                             // 0x3e64442d00000000,
	PI4C := NewFromFloat(2.69515142907905952645e-15)                            // 0x3ce8469898cc5170,
	M4PI := NewFromFloat(1.273239544735162542821171882678754627704620361328125) // 4/pi

	// 将参数转为正数。
	sign := false
	if d.LessThan(NewFromFloat(0.0)) {
		d = d.Neg()
	}

	j := d.Mul(M4PI).IntPart()    // integer part of x/(Pi/4), as integer for tests on the phase angle
	y := NewFromFloat(float64(j)) // integer part of x/(Pi/4), as float

	// 将零值映射到原点。
	if j&1 == 1 {
		j++
		y = y.Add(NewFromFloat(1.0))
	}
	j &= 7 // octant modulo 2Pi radians (360 degrees)
	// 沿 x 轴反射。
	if j > 3 {
		sign = !sign
		j -= 4
	}
	if j > 1 {
		sign = !sign
	}

	z := d.Sub(y.Mul(PI4A)).Sub(y.Mul(PI4B)).Sub(y.Mul(PI4C)) // Extended precision modular arithmetic
	zz := z.Mul(z)

	if j == 1 || j == 2 {
		y = z.Add(z.Mul(zz).Mul(_sin[0].Mul(zz).Add(_sin[1]).Mul(zz).Add(_sin[2]).Mul(zz).Add(_sin[3]).Mul(zz).Add(_sin[4]).Mul(zz).Add(_sin[5])))
	} else {
		w := zz.Mul(zz).Mul(_cos[0].Mul(zz).Add(_cos[1]).Mul(zz).Add(_cos[2]).Mul(zz).Add(_cos[3]).Mul(zz).Add(_cos[4]).Mul(zz).Add(_cos[5]))
		y = NewFromFloat(1.0).Sub(NewFromFloat(0.5).Mul(zz)).Add(w)
	}
	if sign {
		y = y.Neg()
	}
	return y
}

var _tanP = [...]Decimal{
	NewFromFloat(-1.30936939181383777646e+4), // 0xc0c992d8d24f3f38
	NewFromFloat(1.15351664838587416140e+6),  // 0x413199eca5fc9ddd
	NewFromFloat(-1.79565251976484877988e+7), // 0xc1711fead3299176
}
var _tanQ = [...]Decimal{
	NewFromFloat(1.00000000000000000000e+0),
	NewFromFloat(1.36812963470692954678e+4),  //0x40cab8a5eeb36572
	NewFromFloat(-1.32089234440210967447e+6), //0xc13427bc582abc96
	NewFromFloat(2.50083801823357915839e+7),  //0x4177d98fc2ead8ef
	NewFromFloat(-5.38695755929454629881e+7), //0xc189afe03cbe5a31
}

// Tan 返回弧度参数 x 的正切值。
func (d Decimal) Tan() Decimal {

	PI4A := NewFromFloat(7.85398125648498535156e-1)                             // 0x3fe921fb40000000, Pi/4 split into three parts
	PI4B := NewFromFloat(3.77489470793079817668e-8)                             // 0x3e64442d00000000,
	PI4C := NewFromFloat(2.69515142907905952645e-15)                            // 0x3ce8469898cc5170,
	M4PI := NewFromFloat(1.273239544735162542821171882678754627704620361328125) // 4/pi

	if d.Equal(NewFromFloat(0.0)) {
		return d
	}

	// 将参数转为正数，但保留符号。
	sign := false
	if d.LessThan(NewFromFloat(0.0)) {
		d = d.Neg()
		sign = true
	}

	j := d.Mul(M4PI).IntPart()    // integer part of x/(Pi/4), as integer for tests on the phase angle
	y := NewFromFloat(float64(j)) // integer part of x/(Pi/4), as float

	// 将零值映射到原点。
	if j&1 == 1 {
		j++
		y = y.Add(NewFromFloat(1.0))
	}

	z := d.Sub(y.Mul(PI4A)).Sub(y.Mul(PI4B)).Sub(y.Mul(PI4C)) // Extended precision modular arithmetic
	zz := z.Mul(z)

	if zz.GreaterThan(NewFromFloat(1e-14)) {
		w := zz.Mul(_tanP[0].Mul(zz).Add(_tanP[1]).Mul(zz).Add(_tanP[2]))
		x := zz.Add(_tanQ[1]).Mul(zz).Add(_tanQ[2]).Mul(zz).Add(_tanQ[3]).Mul(zz).Add(_tanQ[4])
		y = z.Add(z.Mul(w.Div(x)))
	} else {
		y = z
	}
	if j&2 == 2 {
		y = NewFromFloat(-1.0).Div(y)
	}
	if sign {
		y = y.Neg()
	}
	return y
}
