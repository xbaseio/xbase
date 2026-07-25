// Package xlog configures the process-wide go.uber.org/zap logger.
package xlog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/xbaseio/xbase/etc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultLevel           = "info"
	defaultEncoding        = "console"
	defaultTimeEncoding    = "iso8601"
	defaultStacktraceLevel = "error"
)

var (
	configureMu sync.Mutex
	logger      *zap.Logger
)

func init() {
	if err := registerRotatingSink(); err != nil {
		panic(err)
	}
	if err := configure(); err != nil {
		panic(err)
	}
}

// configure builds a zap logger from the etc.log configuration and installs
// it as zap's process-wide logger.
func configure() error {
	configureMu.Lock()
	defer configureMu.Unlock()

	cfg, buildOpts, err := buildConfig()
	if err != nil {
		return err
	}

	if err = prepareOutputPaths(cfg.OutputPaths); err != nil {
		return err
	}
	if err = prepareOutputPaths(cfg.ErrorOutputPaths); err != nil {
		return err
	}
	if rotation := loadRotationConfig(); rotation.Enabled {
		cfg.OutputPaths = rotatingOutputPaths(cfg.OutputPaths, rotation)
	}

	next, err := cfg.Build(buildOpts...)
	if err != nil {
		return err
	}

	previous := logger
	logger = next
	zap.ReplaceGlobals(next)
	if previous != nil {
		_ = syncLogger(previous)
	}

	return nil
}

// Logger returns the process-wide zap logger installed during package initialization.
func Logger() *zap.Logger {
	return zap.L()
}

// Sugar returns the process-wide sugared logger installed during package initialization.
// Prefer Logger in performance-sensitive code to avoid formatting overhead.
func Sugar() *zap.SugaredLogger {
	return zap.S()
}

// Sync flushes buffered zap output.
func Sync() error {
	configureMu.Lock()
	defer configureMu.Unlock()

	if logger == nil {
		return nil
	}

	return syncLogger(logger)
}

func buildConfig() (zap.Config, []zap.Option, error) {
	level, err := parseLevel(etc.Get("etc.log.level", defaultLevel).String())
	if err != nil {
		return zap.Config{}, nil, err
	}

	encoding := strings.ToLower(etc.Get("etc.log.encoding", defaultEncoding).String())
	if encoding == "text" {
		encoding = "console"
	}
	if encoding != "console" && encoding != "json" {
		return zap.Config{}, nil, errors.New("zap log encoding must be console or json")
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.NameKey = "logger"
	encoderConfig.CallerKey = "caller"
	encoderConfig.FunctionKey = "function"
	encoderConfig.MessageKey = "msg"
	encoderConfig.StacktraceKey = "stacktrace"
	encoderConfig.LineEnding = zapcore.DefaultLineEnding
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	encoderConfig.EncodeName = zapcore.FullNameEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	if encoding == "console" && etc.Get("etc.log.color", true).Bool() {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if etc.Get("etc.log.callerFullPath").Bool() {
		encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	} else {
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	configureTimeEncoder(&encoderConfig)

	outputPaths := etc.Get("etc.log.outputPaths", []string{"stdout"}).Strings()
	if len(outputPaths) == 0 {
		outputPaths = []string{"stdout"}
	}
	errorOutputPaths := etc.Get("etc.log.errorOutputPaths", []string{"stderr"}).Strings()
	if len(errorOutputPaths) == 0 {
		errorOutputPaths = []string{"stderr"}
	}

	cfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       etc.Get("etc.log.development").Bool(),
		DisableCaller:     etc.Get("etc.log.disableCaller").Bool(),
		DisableStacktrace: true,
		Encoding:          encoding,
		EncoderConfig:     encoderConfig,
		OutputPaths:       outputPaths,
		ErrorOutputPaths:  errorOutputPaths,
	}

	if etc.Get("etc.log.sampling.enabled").Bool() {
		cfg.Sampling = &zap.SamplingConfig{
			Initial:    etc.Get("etc.log.sampling.initial", 100).Int(),
			Thereafter: etc.Get("etc.log.sampling.thereafter", 100).Int(),
		}
	}

	buildOpts := make([]zap.Option, 0, 1)
	if !etc.Get("etc.log.disableStacktrace").Bool() {
		stackLevel, parseErr := parseLevel(etc.Get("etc.log.stacktraceLevel", defaultStacktraceLevel).String())
		if parseErr != nil {
			return zap.Config{}, nil, parseErr
		}
		buildOpts = append(buildOpts, zap.AddStacktrace(stackLevel))
	}

	return cfg, buildOpts, nil
}

func configureTimeEncoder(cfg *zapcore.EncoderConfig) {
	switch strings.ToLower(etc.Get("etc.log.timeEncoding", defaultTimeEncoding).String()) {
	case "epoch":
		cfg.EncodeTime = zapcore.EpochTimeEncoder
	case "millis":
		cfg.EncodeTime = zapcore.EpochMillisTimeEncoder
	case "nanos":
		cfg.EncodeTime = zapcore.EpochNanosTimeEncoder
	case "rfc3339nano":
		cfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	default:
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	}
}

func parseLevel(value string) (zapcore.Level, error) {
	var level zapcore.Level
	if err := level.Set(strings.ToLower(value)); err != nil {
		return zapcore.InfoLevel, err
	}
	return level, nil
}

func prepareOutputPaths(paths []string) error {
	for _, path := range paths {
		if path == "" || path == "stdout" || path == "stderr" || strings.Contains(path, "://") {
			continue
		}

		dir := filepath.Dir(path)
		if dir == "." || dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func syncLogger(value *zap.Logger) error {
	err := value.Sync()
	if err == nil || errors.Is(err, os.ErrInvalid) || errors.Is(err, syscall.EINVAL) {
		return nil
	}
	return err
}
