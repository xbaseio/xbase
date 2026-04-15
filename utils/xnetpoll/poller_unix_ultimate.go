//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && poll_opt
// +build darwin dragonfly freebsd linux netbsd openbsd
// +build poll_opt

package xnetpoll

import "unsafe"

func convertPollAttachment(ptr unsafe.Pointer, attachment *PollAttachment) {
	*(**PollAttachment)(ptr) = attachment
}

func restorePollAttachment(ptr unsafe.Pointer) *PollAttachment {
	return *(**PollAttachment)(ptr)
}
