//go:build windows
// +build windows

package stat

import (
	"syscall"
	"time"
)

// CreateTime 获取文件创建时间。
func (fi *fileInfo) CreateTime() time.Time {
	if fi == nil || fi.FileInfo == nil {
		return time.Time{}
	}

	stat, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	if !ok || stat == nil {
		return time.Time{}
	}

	return time.Unix(0, stat.CreationTime.Nanoseconds())
}
