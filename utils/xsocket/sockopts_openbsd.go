package xsocket

import "github.com/xbaseio/xbase/xerrors"

// SetKeepAlivePeriod 在 OpenBSD 上未实现，
// 因为没有类似 Linux 的 TCP_KEEPIDLE、TCP_KEEPINTVL 和 TCP_KEEPCNT。
func SetKeepAlivePeriod(_, _ int) error {
	// OpenBSD 不支持按 socket 设置 TCP keepalive 相关参数。
	return xerrors.ErrUnsupportedOp
}

// SetKeepAlive 在 OpenBSD 上未实现。
func SetKeepAlive(_ int, _ bool, _, _, _ int) error {
	// OpenBSD 不支持按 socket 设置 TCP keepalive 相关参数。
	return xerrors.ErrUnsupportedOp
}
