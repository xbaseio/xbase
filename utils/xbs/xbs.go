package xbs

import (
	"unsafe"
)

// BytesToString 将 byte 切片转换为 string，不发生内存拷贝（零拷贝）。
//
// 注意：
// 返回的 string 与原始 byte 切片共享底层内存，
// 如果后续修改 b，会导致 string 内容被篡改（违反 string 不可变语义）。
func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// StringToBytes 将 string 转换为 byte 切片，不发生内存拷贝（零拷贝）。
//
// 注意：
// 返回的 []byte 与原始 string 共享底层内存，
// 对返回切片的写操作会导致未定义行为（因为 string 在 Go 中是只读的）。
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
