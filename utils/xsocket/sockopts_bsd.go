//go:build dragonfly || freebsd || netbsd || openbsd
// +build dragonfly freebsd netbsd openbsd

package xsocket

import "github.com/xbaseio/xbase/xerrors"

// SetBindToDevice 在 *BSD 系统上未实现，
// 因为没有类似 Linux 的 SO_BINDTODEVICE 功能。
func SetBindToDevice(_ int, _ string) error {
	return xerrors.ErrUnsupportedOp
}
