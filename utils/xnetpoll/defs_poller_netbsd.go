//go:build netbsd
// +build netbsd

package xnetpoll

// IOEvent 表示 NetBSD 平台下的 I/O 事件类型。
type IOEvent = uint32

// IOFlags 表示 NetBSD 平台下的 I/O 事件标志位类型。
type IOFlags = uint32
