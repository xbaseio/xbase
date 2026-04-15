//go:build !windows && !freebsd && !darwin && !linux && !openbsd
// +build !windows,!freebsd,!darwin,!linux,!openbsd

package stat

import "time"

// CreateTime 获取文件创建时间。
// 其他平台下如果没有专门实现，这里先退化为修改时间。
func (fi *fileInfo) CreateTime() time.Time {
	if fi == nil || fi.FileInfo == nil {
		return time.Time{}
	}

	return fi.ModTime()
}
