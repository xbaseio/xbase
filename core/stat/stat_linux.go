//go:build linux
// +build linux

package stat

import "time"

// CreateTime 获取文件创建时间。
// Linux 下没有统一可靠的创建时间，这里退化为修改时间。
func (fi *fileInfo) CreateTime() time.Time {
	if fi == nil || fi.FileInfo == nil {
		return time.Time{}
	}

	return fi.ModifyTime()
}
