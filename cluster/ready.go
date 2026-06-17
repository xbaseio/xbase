package cluster

import (
	"fmt"
	"net"
	"time"
)

func WaitForTCPListen(addr string, timeout time.Duration) error {
	target, err := normalizeDialAddr(addr)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for {
		conn, dialErr := net.DialTimeout("tcp", target, 100*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("dial %s: %w", target, dialErr)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func normalizeDialAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	return net.JoinHostPort(host, port), nil
}
