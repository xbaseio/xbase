package net

import (
	"testing"
	"time"

	"github.com/xbaseio/xbase/xerrors"
)

func TestResolveFirstPublicIPTimeoutDoesNotPanicAfterReturn(t *testing.T) {
	query := func(url string, timeout time.Duration) (string, error) {
		time.Sleep(timeout + 20*time.Millisecond)
		return "1.1.1.1", nil
	}

	if _, err := resolveFirstPublicIP([]string{"slow-1", "slow-2"}, query, 10*time.Millisecond); err == nil {
		t.Fatal("expected timeout error")
	}

	time.Sleep(50 * time.Millisecond)
}

func TestResolveFirstPublicIPReturnsFirstSuccess(t *testing.T) {
	query := func(url string, timeout time.Duration) (string, error) {
		switch url {
		case "slow":
			time.Sleep(20 * time.Millisecond)
			return "2.2.2.2", nil
		case "fast":
			time.Sleep(5 * time.Millisecond)
			return " 1.1.1.1 \n", nil
		default:
			return "", errTestNotFoundIP
		}
	}

	ip, err := resolveFirstPublicIP([]string{"slow", "fast"}, query, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ip != "1.1.1.1" {
		t.Fatalf("unexpected ip: %q", ip)
	}
}

var errTestNotFoundIP = xerrors.ErrNotFoundIPAddress
