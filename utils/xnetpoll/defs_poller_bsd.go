//go:build darwin || dragonfly || freebsd || openbsd
// +build darwin dragonfly freebsd openbsd

package xnetpoll

// IOFlags 表示 I/O 事件的标志位类型（BSD）。
type IOFlags = uint16

// IOEvent 表示 I/O 事件类型（BSD）。
type IOEvent = int16
