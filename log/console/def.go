package console

import "github.com/xbaseio/xbase/log/internal"

// Format 日志输出格式
type Format = internal.Format

const (
	FormatText = internal.FormatText // 文本格式
	FormatJson = internal.FormatJson // JSON格式
)
