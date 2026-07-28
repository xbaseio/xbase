package xlog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xbaseio/xbase/config"
	"github.com/xbaseio/xbase/etc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitializedLogger(t *testing.T) {
	if logger == nil {
		t.Fatal("log package init did not configure a logger")
	}
	if Logger() != zap.L() {
		t.Fatal("Logger does not return the initialized global logger")
	}
}

func TestConfigureRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
	}{
		{
			name: "level",
			values: map[string]any{
				"level": "verbose",
			},
		},
		{
			name: "encoding",
			values: map[string]any{
				"encoding": "xml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestConfig(t, tt.values)
			if err := configure(); err == nil {
				t.Fatalf("configure() accepted invalid %s", tt.name)
			}
		})
	}
}

func TestConfigureInstallsGlobalLogger(t *testing.T) {
	setTestConfig(t, map[string]any{
		"level":             "debug",
		"encoding":          "json",
		"outputPaths":       []string{"stdout"},
		"errorOutputPaths":  []string{"stderr"},
		"disableStacktrace": true,
	})

	previous := zap.L()
	if err := configure(); err != nil {
		t.Fatalf("configure() error = %v", err)
	}
	if zap.L() == previous {
		t.Fatal("configure() did not replace zap's global logger")
	}
	if Logger() != zap.L() {
		t.Fatal("Logger() does not return zap's configured global logger")
	}
	if !Logger().Core().Enabled(zapcore.DebugLevel) {
		t.Fatal("configured logger does not enable debug level")
	}
	if err := Sync(); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
}

func TestPrepareOutputPathsCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	if err := prepareOutputPaths([]string{filepath.Join(dir, "xbase.log")}); err != nil {
		t.Fatalf("prepareOutputPaths() error = %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("prepared path %q is not a directory", dir)
	}
}

func TestApplyRotationIncludesErrorOutputFiles(t *testing.T) {
	cfg := zap.Config{
		OutputPaths:      []string{"stdout", "app.log"},
		ErrorOutputPaths: []string{"stderr", "zap-errors.log"},
	}
	applyRotation(&cfg, rotationConfig{Daily: true, MaxSize: 100})

	if cfg.OutputPaths[0] != "stdout" || cfg.OutputPaths[1] == "app.log" {
		t.Fatalf("output paths were not rotated correctly: %v", cfg.OutputPaths)
	}
	if cfg.ErrorOutputPaths[0] != "stderr" || cfg.ErrorOutputPaths[1] == "zap-errors.log" {
		t.Fatalf("error output paths were not rotated correctly: %v", cfg.ErrorOutputPaths)
	}
}

func TestBuildLoggerDoesNotWriteColorCodesToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xbase.log")
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg := zap.Config{
		Level:         zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Encoding:      "console",
		EncoderConfig: encoderConfig,
		OutputPaths: rotatingOutputPaths([]string{path}, rotationConfig{
			Daily:   true,
			MaxSize: 100,
		}),
		ErrorOutputPaths: []string{"stderr"},
	}

	value, err := buildLogger(cfg)
	if err != nil {
		t.Fatalf("buildLogger() error = %v", err)
	}
	value.Info("plain file output")
	if err = syncLogger(value); err != nil {
		t.Fatalf("logger Sync() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if bytes.Contains(content, []byte("\x1b[")) {
		t.Fatalf("file log contains ANSI color code: %q", content)
	}
}

func setTestConfig(t *testing.T, values map[string]any) {
	t.Helper()

	configurator := config.NewConfigurator()
	etc.SetConfigurator(configurator)
	if err := etc.Set("etc.log", values); err != nil {
		t.Fatalf("set log config: %v", err)
	}
}
