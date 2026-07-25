package logger

import (
	"sync"

	rpcxlog "github.com/smallnest/rpcx/log"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var once sync.Once

func InitLogger() {
	once.Do(func() {
		rpcxlog.SetLogger(&logger{
			level:  zapcore.ErrorLevel,
			logger: xlog.Sugar(),
		})
	})
}

type logger struct {
	level  zapcore.Level
	logger *zap.SugaredLogger
}

func (l *logger) Debug(v ...any) {
	if l.level <= zapcore.DebugLevel {
		l.logger.Debug(v...)
	}
}

func (l *logger) Debugf(format string, v ...any) {
	if l.level <= zapcore.DebugLevel {
		l.logger.Debugf(format, v...)
	}
}

func (l *logger) Info(v ...any) {
	if l.level <= zapcore.InfoLevel {
		l.logger.Info(v...)
	}
}

func (l *logger) Infof(format string, v ...any) {
	if l.level <= zapcore.InfoLevel {
		l.logger.Infof(format, v...)
	}
}

func (l *logger) Warn(v ...any) {
	if l.level <= zapcore.WarnLevel {
		l.logger.Warn(v...)
	}
}

func (l *logger) Warnf(format string, v ...any) {
	if l.level <= zapcore.WarnLevel {
		l.logger.Warnf(format, v...)
	}
}

func (l *logger) Error(v ...any) {
	if l.level <= zapcore.ErrorLevel {
		l.logger.Error(v...)
	}
}

func (l *logger) Errorf(format string, v ...any) {
	if l.level <= zapcore.ErrorLevel {
		l.logger.Errorf(format, v...)
	}
}

// rpcx fatal and panic hooks are treated as error records. Lifecycle control
// remains with xbase rather than a third-party logging callback.
func (l *logger) Fatal(v ...any)                 { l.Error(v...) }
func (l *logger) Fatalf(format string, v ...any) { l.Errorf(format, v...) }
func (l *logger) Panic(v ...any)                 { l.Error(v...) }
func (l *logger) Panicf(format string, v ...any) { l.Errorf(format, v...) }
