package stat

import (
	"os"
	"time"
)

// FileInfo 扩展标准库 os.FileInfo。
type FileInfo interface {
	os.FileInfo

	// CreateTime 获取文件创建时间。
	CreateTime() time.Time

	// ModifyTime 获取文件修改时间。
	ModifyTime() time.Time

	// IsFile 判断是否为普通文件。
	IsFile() bool
}

// fileInfo 是 FileInfo 的默认实现。
type fileInfo struct {
	os.FileInfo
}

// ModifyTime 获取文件修改时间。
func (fi *fileInfo) ModifyTime() time.Time {
	if fi == nil || fi.FileInfo == nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// IsFile 判断是否为普通文件。
func (fi *fileInfo) IsFile() bool {
	if fi == nil || fi.FileInfo == nil {
		return false
	}
	return fi.Mode().IsRegular()
}
