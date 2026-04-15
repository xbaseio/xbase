//go:build openbsd
// +build openbsd

package stat

import (
	"syscall"
	"time"
)

// CreateTime 获取文件“创建时间”。
// 注意：OpenBSD 不支持真正的创建时间，这里退化为 ctime。
func (fs *fileInfo) CreateTime() time.Time {
	if fs == nil || fs.FileInfo == nil {
		return time.Time{}
	}

	stat, ok := fs.FileInfo.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return time.Time{}
	}

	return time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
}
