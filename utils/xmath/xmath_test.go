package xmath

import "testing"

// TestCeilToPowerOfTwo 测试 CeilToPowerOfTwo 的各种边界与典型输入。
func TestCeilToPowerOfTwo(t *testing.T) {
	type args struct {
		n int
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		// 边界值测试：0、1、2
		{name: "zero", args: args{n: 0}, want: 2},
		{name: "one", args: args{n: 1}, want: 2},
		{name: "two", args: args{n: 2}, want: 2},

		// 小范围测试：3-15
		{name: "three", args: args{n: 3}, want: 1 << 2},
		{name: "four", args: args{n: 4}, want: 1 << 2},
		{name: "five", args: args{n: 5}, want: 1 << 3},
		{name: "six", args: args{n: 6}, want: 1 << 3},
		{name: "seven", args: args{n: 7}, want: 1 << 3},
		{name: "eight", args: args{n: 8}, want: 1 << 3},
		{name: "nine", args: args{n: 9}, want: 1 << 4},
		{name: "ten", args: args{n: 10}, want: 1 << 4},
		{name: "fifteen", args: args{n: 15}, want: 1 << 4},

		// 已经是 2 的幂的情况
		{name: "power_of_two_16", args: args{n: 1 << 4}, want: 1 << 4},
		{name: "power_of_two_32", args: args{n: 1 << 5}, want: 1 << 5},
		{name: "power_of_two_64", args: args{n: 1 << 6}, want: 1 << 6},
		{name: "power_of_two_128", args: args{n: 1 << 7}, want: 1 << 7},
		{name: "power_of_two_256", args: args{n: 1 << 8}, want: 1 << 8},
		{name: "power_of_two_512", args: args{n: 1 << 9}, want: 1 << 9},
		{name: "power_of_two_1024", args: args{n: 1 << 10}, want: 1 << 10},

		// 接近 2 的幂的情况（上下边界）
		{name: "near_power_17", args: args{n: (1 << 4) + 1}, want: 1 << 5},
		{name: "near_power_31", args: args{n: (1 << 5) - 1}, want: 1 << 5},
		{name: "near_power_33", args: args{n: (1 << 5) + 1}, want: 1 << 6},
		{name: "near_power_63", args: args{n: (1 << 6) - 1}, want: 1 << 6},
		{name: "near_power_65", args: args{n: (1 << 6) + 1}, want: 1 << 7},
		{name: "near_power_127", args: args{n: (1 << 7) - 1}, want: 1 << 7},
		{name: "near_power_129", args: args{n: (1 << 7) + 1}, want: 1 << 8},
		{name: "near_power_255", args: args{n: (1 << 8) - 1}, want: 1 << 8},
		{name: "near_power_257", args: args{n: (1 << 8) + 1}, want: 1 << 9},
		{name: "near_power_511", args: args{n: (1 << 9) - 1}, want: 1 << 9},
		{name: "near_power_513", args: args{n: (1 << 9) + 1}, want: 1 << 10},
		{name: "near_power_1023", args: args{n: (1 << 10) - 1}, want: 1 << 10},

		// 中等规模测试
		{name: "medium_100", args: args{n: 100}, want: 1 << 7},
		{name: "medium_200", args: args{n: 200}, want: 1 << 8},
		{name: "medium_500", args: args{n: 500}, want: 1 << 9},
		{name: "medium_1000", args: args{n: 1000}, want: 1 << 10},
		{name: "medium_2000", args: args{n: 2000}, want: 1 << 11},
		{name: "medium_5000", args: args{n: 5000}, want: 1 << 13},
		{name: "medium_10000", args: args{n: 10000}, want: 1 << 14},

		// 较大值（2^10 附近）
		{name: "large_1024_minus_1", args: args{n: 1<<10 - 1}, want: 1 << 10},
		{name: "large_1024", args: args{n: 1 << 10}, want: 1 << 10},
		{name: "large_1024_plus_1", args: args{n: 1<<10 + 1}, want: 1 << 11},
		{name: "large_2047", args: args{n: (1 << 11) - 1}, want: 1 << 11},
		{name: "large_2048", args: args{n: 1 << 11}, want: 1 << 11},
		{name: "large_2049", args: args{n: (1 << 11) + 1}, want: 1 << 12},

		// 很大值（2^20 附近）
		{name: "very_large_1M_minus_1", args: args{n: 1<<20 - 1}, want: 1 << 20},
		{name: "very_large_1M", args: args{n: 1 << 20}, want: 1 << 20},
		{name: "very_large_1M_plus_1", args: args{n: 1<<20 + 1}, want: 1 << 21},

		// 极大值（32 位边界）
		{name: "huge_1G_minus_1", args: args{n: 1<<30 - 1}, want: 1 << 30},
		{name: "huge_1G", args: args{n: 1 << 30}, want: 1 << 30},
		{name: "huge_1G_plus_1", args: args{n: 1<<30 + 1}, want: 1 << 31},

		// 64 位环境测试（2^32 附近）
		{name: "extreme_2_32_minus_1", args: args{n: 1<<32 - 1}, want: 1 << 32},
		{name: "extreme_2_32", args: args{n: 1 << 32}, want: 1 << 32},
		{name: "extreme_2_32_plus_1", args: args{n: 1<<32 + 1}, want: 1 << 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CeilToPowerOfTwo(tt.args.n); got != tt.want {
				t.Errorf("CeilToPowerOfTwo() = %v, want %v", got, tt.want)
			}
		})
	}
}
