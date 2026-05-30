package registry_test

import (
	"testing"

	"github.com/xbaseio/xbase/registry"
)

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2", "1", 1},
		{"1", "2", -1},
		{"10", "2", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0", "1.0.0", 0},
		{"", "1", -1},
		{"1", "1", 0},
	}

	for _, c := range cases {
		got := registry.CompareVersion(c.a, c.b)
		if got != c.want {
			t.Fatalf("CompareVersion(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
