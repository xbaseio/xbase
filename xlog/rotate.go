package xlog

import (
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xbaseio/xbase/etc"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

const rotatingSinkScheme = "xlogrotate"

type rotationConfig struct {
	Enabled    bool
	Daily      bool
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
	LocalTime  bool
}

type rotatingSink struct {
	mu        sync.Mutex
	writer    *lumberjack.Logger
	daily     bool
	localTime bool
	day       int
	now       func() time.Time
}

func registerRotatingSink() error {
	return zap.RegisterSink(rotatingSinkScheme, newRotatingSink)
}

func loadRotationConfig() rotationConfig {
	return rotationConfig{
		Enabled:    etc.Get("etc.log.rotation.enabled", true).Bool(),
		Daily:      etc.Get("etc.log.rotation.daily", true).Bool(),
		MaxSize:    etc.Get("etc.log.rotation.maxSize", 100).Int(),
		MaxAge:     etc.Get("etc.log.rotation.maxAge", 7).Int(),
		MaxBackups: etc.Get("etc.log.rotation.maxBackups", 30).Int(),
		Compress:   etc.Get("etc.log.rotation.compress", true).Bool(),
		LocalTime:  etc.Get("etc.log.rotation.localTime", true).Bool(),
	}
}

func rotatingOutputPaths(paths []string, cfg rotationConfig) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		if !isFilePath(path) {
			result[i] = path
			continue
		}

		values := url.Values{}
		values.Set("daily", strconv.FormatBool(cfg.Daily))
		values.Set("maxSize", strconv.Itoa(cfg.MaxSize))
		values.Set("maxAge", strconv.Itoa(cfg.MaxAge))
		values.Set("maxBackups", strconv.Itoa(cfg.MaxBackups))
		values.Set("compress", strconv.FormatBool(cfg.Compress))
		values.Set("localTime", strconv.FormatBool(cfg.LocalTime))
		result[i] = rotatingSinkScheme + ":" + base64.RawURLEncoding.EncodeToString([]byte(path)) + "?" + values.Encode()
	}
	return result
}

func newRotatingSink(target *url.URL) (zap.Sink, error) {
	if target.Scheme != rotatingSinkScheme || target.Opaque == "" {
		return nil, errors.New("invalid rotating log sink URL")
	}

	path, err := base64.RawURLEncoding.DecodeString(target.Opaque)
	if err != nil {
		return nil, err
	}

	values := target.Query()
	cfg := rotationConfig{
		Daily:      queryBool(values, "daily"),
		MaxSize:    queryInt(values, "maxSize"),
		MaxAge:     queryInt(values, "maxAge"),
		MaxBackups: queryInt(values, "maxBackups"),
		Compress:   queryBool(values, "compress"),
		LocalTime:  queryBool(values, "localTime"),
	}
	if cfg.MaxSize <= 0 {
		return nil, errors.New("log rotation maxSize must be greater than zero")
	}

	return newRotatingFileSink(string(path), cfg), nil
}

func newRotatingFileSink(path string, cfg rotationConfig) *rotatingSink {
	sink := &rotatingSink{
		writer: &lumberjack.Logger{
			Filename:   path,
			MaxSize:    cfg.MaxSize,
			MaxAge:     cfg.MaxAge,
			MaxBackups: cfg.MaxBackups,
			LocalTime:  cfg.LocalTime,
			Compress:   cfg.Compress,
		},
		daily:     cfg.Daily,
		localTime: cfg.LocalTime,
		now:       time.Now,
	}
	sink.day = sink.currentDay()
	if info, err := os.Stat(path); err == nil {
		sink.day = sink.dayOf(info.ModTime())
	}
	return sink
}

func (s *rotatingSink) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	day := s.currentDay()
	if s.daily && day != s.day {
		if err := s.writer.Rotate(); err != nil {
			return 0, err
		}
		s.day = day
	}

	return s.writer.Write(data)
}

// Sync closes the active file under the same lock as Write. Lumberjack opens
// it again on the next write, so Sync is safe both at shutdown and at runtime.
func (s *rotatingSink) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writer.Close()
}

func (s *rotatingSink) Close() error {
	return s.Sync()
}

func (s *rotatingSink) currentDay() int {
	return s.dayOf(s.now())
}

func (s *rotatingSink) dayOf(value time.Time) int {
	now := value
	if !s.localTime {
		now = now.UTC()
	}
	year, month, day := now.Date()
	return year*10000 + int(month)*100 + day
}

func isFilePath(path string) bool {
	return path != "" && path != "stdout" && path != "stderr" && !strings.Contains(path, "://")
}

func queryBool(values url.Values, key string) bool {
	value, _ := strconv.ParseBool(values.Get(key))
	return value
}

func queryInt(values url.Values, key string) int {
	value, _ := strconv.Atoi(values.Get(key))
	return value
}
