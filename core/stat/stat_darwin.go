//go:build darwin
// +build darwin

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

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return time.Time{}
	}

	return time.Unix(
		int64(stat.Birthtimespec.Sec),
		int64(stat.Birthtimespec.Nsec),
	)
}
