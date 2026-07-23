package mode

import (
	"github.com/xbaseio/xbase/env"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/flag"
)

const (
	xbaseModeEtcName = "etc.mode"
	xbaseModeArgName = "mode"
	xbaseModeEnvName = "XBASE_MODE"
)

const (
	// DebugMode indicates xbase mode is debug.
	DebugMode = "debug"
	// DevelopMode indicates xbase mode is develop.
	DevelopMode = "develop"
	// ReleaseMode indicates xbase mode is release.
	ReleaseMode = "release"
	// TestMode indicates xbase mode is test.
	TestMode = "test"
)

var xbaseMode string

// 优先级： 配置文件 < 环境变量 < 运行参数 < mode.SetMode()
func init() {
	mode := etc.Get(xbaseModeEtcName, DebugMode).String()
	mode = env.Get(xbaseModeEnvName, mode).String()
	mode = flag.String(xbaseModeArgName, mode)
	SetMode(mode)
}

// SetMode 设置运行模式
func SetMode(m string) {
	if m == "" {
		m = DebugMode
	}

	switch m {
	case DebugMode, DevelopMode, TestMode, ReleaseMode:
		xbaseMode = m
	default:
		panic("xbase mode unknown: " + m + " (available mode: debug develop test release)")
	}
}

// GetMode 获取运行模式
func GetMode() string {
	return xbaseMode
}

// IsDebugMode 是否Debug模式
func IsDebugMode() bool {
	return xbaseMode == DebugMode
}

// IsDevelopMode 是否Develop模式
func IsDevelopMode() bool {
	return xbaseMode == DevelopMode
}

// IsTestMode 是否Test模式
func IsTestMode() bool {
	return xbaseMode == TestMode
}

// IsReleaseMode 是否Release模式
func IsReleaseMode() bool {
	return xbaseMode == ReleaseMode
}
