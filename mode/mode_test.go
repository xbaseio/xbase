package mode_test

import (
	"flag"
	"testing"

	"github.com/xbaseio/xbase/mode"
)

func TestGetMode(t *testing.T) {
	flag.Parse()

	t.Log(mode.GetMode())
}

func TestSetMode(t *testing.T) {
	original := mode.GetMode()
	t.Cleanup(func() { mode.SetMode(original) })

	tests := []struct {
		value string
		check func() bool
	}{
		{value: mode.DebugMode, check: mode.IsDebugMode},
		{value: mode.DevelopMode, check: mode.IsDevelopMode},
		{value: mode.TestMode, check: mode.IsTestMode},
		{value: mode.ReleaseMode, check: mode.IsReleaseMode},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			mode.SetMode(tt.value)
			if got := mode.GetMode(); got != tt.value {
				t.Fatalf("GetMode() = %q, want %q", got, tt.value)
			}
			if !tt.check() {
				t.Fatalf("mode check failed for %q", tt.value)
			}
		})
	}
}
