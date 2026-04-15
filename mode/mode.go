package mode

import (
	"github.com/xbaseio/xbase/env"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/flag"
)

const (
	dueModeEtcName = "etc.mode"
	dueModeArgName = "mode"
	dueModeEnvName = "DUE_MODE"
)

const (
	// DebugMode indicates xbase mode is debug.
	DebugMode = "debug"
	// ReleaseMode indicates xbase mode is release.
	ReleaseMode = "release"
	// TestMode indicates xbase mode is test.
	TestMode = "test"
)

var dueMode string

// 优先级： 配置文件 < 环境变量 < 运行参数 < mode.SetMode()
func init() {
	mode := etc.Get(dueModeEtcName, DebugMode).String()
	mode = env.Get(dueModeEnvName, mode).String()
	mode = flag.String(dueModeArgName, mode)
	SetMode(mode)
}

// SetMode 设置运行模式
func SetMode(m string) {
	if m == "" {
		m = DebugMode
	}

	switch m {
	case DebugMode, TestMode, ReleaseMode:
		dueMode = m
	default:
		panic("xbase mode unknown: " + m + " (available mode: debug test release)")
	}
}

// GetMode 获取运行模式
func GetMode() string {
	return dueMode
}

// IsDebugMode 是否Debug模式
func IsDebugMode() bool {
	return dueMode == DebugMode
}

// IsTestMode 是否Test模式
func IsTestMode() bool {
	return dueMode == TestMode
}

// IsReleaseMode 是否Release模式
func IsReleaseMode() bool {
	return dueMode == ReleaseMode
}
