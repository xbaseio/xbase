// 版权所有 2009 Go Authors。保留所有权利。
// 本源码的使用遵循 BSD 风格许可证，许可证可在 LICENSE 文件中找到。

// 多精度十进制数。
//
// 注意：这里只用于浮点数格式化，不是通用 decimal 库。
//
// 支持的操作只有：赋值、二进制左移、二进制右移。
//
// 为什么可以精确处理二进制浮点数到多精度十进制？
// 因为 2 可以整除 10，所以二进制浮点转十进制可以精确表达。
//
// 但是反过来不行：
// 十进制浮点不能总是用二进制精确表达。
package decimal

type floatInfo struct {
	mantbits uint // 尾数位数
	expbits  uint // 指数位数
	bias     int  // 指数偏移
}

var float32info = floatInfo{23, 8, -127}
var float64info = floatInfo{52, 11, -1023}

// roundShortest 会把 d，也就是 mant * 2^exp，舍入成最短的十进制数字。
// 这个最短数字必须满足：重新解析回浮点数时，能够精确还原出原来的浮点值。
func roundShortest(d *decimal, mant uint64, exp int, flt *floatInfo) {
	// 如果尾数是 0，说明这个数就是 0，直接结束。
	if mant == 0 {
		d.nd = 0
		return
	}

	// 计算上界和下界。
	//
	// 任何处在 lower 和 upper 之间的十进制数，
	// 在舍入回浮点数时，都可能得到原来的浮点值。
	//
	// 也就是说，我们要找的最短十进制表示，
	// 必须落在这个可还原范围内。

	// 有些情况下，可以直接判断当前数字已经是最短表示。
	//
	// 假设 d 不是非规格化数，因此：
	//     2^exp <= d < 10^dp
	//
	// 最接近的更短数字，至少距离当前数 10^(dp-nd)。
	//
	// 后面计算出的上下边界，距离当前数最多是：
	//     2^(exp-mantbits)
	//
	// 所以如果：
	//     10^(dp-nd) > 2^(exp-mantbits)
	//
	// 当前数字就已经是最短的。
	//
	// 等价转换成 log2：
	//     log2(10) * (dp-nd) > exp-mantbits
	//
	// 因为 log2(10) > 3.32，
	// 所以可以用整数近似：
	//     332/100 * (dp-nd) >= exp-mantbits
	minexp := flt.bias + 1 // 最小可能指数
	if exp > minexp && 332*(d.dp-d.nd) >= 100*(exp-int(flt.mantbits)) {
		// 当前数字已经是最短表示。
		return
	}

	// d = mant << (exp - mantbits)
	//
	// 下一个更大的浮点数是：
	//     (mant + 1) << (exp - mantbits)
	//
	// 上边界是它们的中点：
	//     (mant*2 + 1) << (exp - mantbits - 1)
	upper := new(decimal)
	upper.Assign(mant*2 + 1)
	upper.Shift(exp - int(flt.mantbits) - 1)

	// d = mant << (exp - mantbits)
	//
	// 下一个更小的浮点数通常是：
	//     (mant - 1) << (exp - mantbits)
	//
	// 但是有一种特殊情况：
	// 如果 mant-1 会导致有效位掉下去，并且 exp 不是最小指数，
	// 那么下一个更小的数是：
	//     (mant*2 - 1) << (exp - mantbits - 1)
	//
	// 不管是哪种情况，都统一称为：
	//     mantlo << (explo - mantbits)
	//
	// 下边界是它们的中点：
	//     (mantlo*2 + 1) << (explo - mantbits - 1)
	var mantlo uint64
	var explo int
	if mant > 1<<flt.mantbits || exp == minexp {
		mantlo = mant - 1
		explo = exp
	} else {
		mantlo = mant*2 - 1
		explo = exp - 1
	}

	lower := new(decimal)
	lower.Assign(mantlo*2 + 1)
	lower.Shift(explo - int(flt.mantbits) - 1)

	// 只有当原始尾数 mant 是偶数时，
	// upper 和 lower 边界值本身才是允许的结果。
	//
	// 这是因为 IEEE 浮点数使用 round-to-even，也就是“四舍六入五成双”。
	// 如果 mant 是偶数，正好落在边界上时会舍入回原来的 mant，
	// 而不是舍入到相邻的浮点数。
	inclusive := mant%2 == 0

	// 遍历十进制数字时，需要判断“向上舍入”是否还在 upper 范围内。
	//
	// upperdelta 用来跟踪 d 和 upper 之间的差异：
	//
	// upperdelta == 0：
	//     到目前为止，d 和 upper 的数字完全相同。
	//
	// upperdelta == 1：
	//     之前某一位 d 和 upper 相差 1，
	//     并且后续 d 一直是 9，upper 一直是 0。
	//
	//     例如：
	//         d     = 12345999...
	//         upper = 12346000...
	//
	//     这种情况下，如果 upper 是排他的，
	//     向上舍入可能会越过边界。
	//
	// upperdelta == 2：
	//     d 和 upper 的差距已经大于 1。
	//     此时可以确定，向上舍入仍然落在 upper 范围内。
	var upperdelta uint8

	// 现在开始计算最少需要多少位十进制数字。
	//
	// 从高位到低位扫描，直到 d 能够和 lower / upper 区分开。
	for ui := 0; ; ui++ {
		// lower、d、upper 的小数点位置可能不一样。
		//
		// 这里 upper 是最长的，
		// 所以从 ui == 0 开始遍历 upper。
		//
		// li 和 mi 是 lower、d 中对应的数字下标，
		// 它们可能从 -1 开始。
		mi := ui - upper.dp + d.dp
		if mi >= d.nd {
			break
		}

		li := ui - upper.dp + lower.dp

		l := byte('0') // lower 当前位数字
		if li >= 0 && li < lower.nd {
			l = lower.d[li]
		}

		m := byte('0') // d 当前位数字
		if mi >= 0 {
			m = d.d[mi]
		}

		u := byte('0') // upper 当前位数字
		if ui < upper.nd {
			u = upper.d[ui]
		}

		// 判断是否可以向下舍入，也就是直接截断。
		//
		// 可以向下舍入的情况：
		// 1. lower 当前位和 d 当前位不同，说明截断后仍然大于 lower；
		// 2. lower 是包含边界，并且当前正好到达 lower 的最后一位。
		okdown := l != m || inclusive && li+1 == lower.nd

		switch {
		case upperdelta == 0 && m+1 < u:
			// 例如：
			//     m = 12345xxx
			//     u = 12347xxx
			//
			// 差距已经大于 1，
			// 所以向上舍入肯定不会超过 upper。
			upperdelta = 2

		case upperdelta == 0 && m != u:
			// 例如：
			//     m = 12345xxx
			//     u = 12346xxx
			//
			// 当前位刚好差 1，
			// 后续还需要继续观察。
			upperdelta = 1

		case upperdelta == 1 && (m != '9' || u != '0'):
			// 例如：
			//     m = 1234598x
			//     u = 1234600x
			//
			// 已经不再是 d 后续全 9、upper 后续全 0 的临界状态，
			// 因此可以确定向上舍入不会超过 upper。
			upperdelta = 2
		}

		// 判断是否可以向上舍入。
		//
		// 可以向上舍入的条件：
		// 1. upper 和 d 已经出现差异；
		// 2. 并且满足以下任意一种：
		//    - upper 是包含边界；
		//    - upper 比向上舍入结果更大；
		//    - 当前还没有到 upper 的最后一位。
		okup := upperdelta > 0 && (inclusive || upperdelta > 1 || ui+1 < upper.nd)

		// 如果向下和向上都可以，
		// 就选择离原值最近的方式进行舍入。
		//
		// 如果只有一种方式可以，就使用那一种。
		switch {
		case okdown && okup:
			d.Round(mi + 1)
			return

		case okdown:
			d.RoundDown(mi + 1)
			return

		case okup:
			d.RoundUp(mi + 1)
			return
		}
	}
}
