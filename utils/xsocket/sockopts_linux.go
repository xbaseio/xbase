package xsocket

import (
	"os"

	"golang.org/x/sys/unix"
)

// SetBindToDevice 将 socket 绑定到指定的网络接口。
//
// Linux 上的 SO_BINDTODEVICE 是双向生效的：
// 既只接收来自该接口的数据包，也只通过该接口发送数据，
// 而不会走系统默认路由。
func SetBindToDevice(fd int, ifname string) error {
	return os.NewSyscallError("setsockopt",
		unix.BindToDevice(fd, ifname))
}
