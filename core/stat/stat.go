package stat

import "os"

// Stat 获取文件信息。
func Stat(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: info}, nil
}

// Lstat 获取文件信息，但不跟随符号链接。
func Lstat(path string) (FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: info}, nil
}

// MustStat 获取文件信息，失败返回 nil。
func MustStat(path string) FileInfo {
	info, err := Stat(path)
	if err != nil {
		return nil
	}
	return info
}

// Exists 判断路径是否存在。
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir 判断路径是否为目录。
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile 判断路径是否为普通文件。
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}
