package xsocket

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func maxListenerBacklog() int {
	fd, err := os.Open("/proc/sys/net/core/somaxconn")
	if err != nil {
		return unix.SOMAXCONN
	}
	defer fd.Close() //nolint:errcheck

	rd := bufio.NewReader(fd)
	line, err := rd.ReadString('\n')
	if err != nil {
		return unix.SOMAXCONN
	}

	f := strings.Fields(line)
	if len(f) < 1 {
		return unix.SOMAXCONN
	}

	n, err := strconv.Atoi(f[0])
	if err != nil || n == 0 {
		return unix.SOMAXCONN
	}

	// Linux 使用 uint16 存储 backlog。
	// 这里对值进行截断以避免溢出回绕。
	// 参考 issue 5030。
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}

	return n
}
