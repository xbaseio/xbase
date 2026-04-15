//go:build (darwin || dragonfly || freebsd || netbsd || openbsd) && !(386 || arm || mips || mipsle)
// +build darwin dragonfly freebsd netbsd openbsd
// +build !386
// +build !arm
// +build !mips
// +build !mipsle

package xnetpoll

// keventIdent 是 kqueue 在 BSD/macOS 非 32 位系统下使用的事件标识类型。
type keventIdent = uint64
