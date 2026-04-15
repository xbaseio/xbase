package log_test

import (
	"testing"

	"github.com/xbaseio/xbase/log"
)

func TestLog(t *testing.T) {
	logger := log.NewLogger()

	logger.Debug("welcome to xbase-framework")
	logger.Info("welcome to xbase-framework")
	logger.Warn("welcome to xbase-framework")
	logger.Error("welcome to xbase-framework")

}

func TestLogger(t *testing.T) {
	log.SetLogger(log.NewLogger(log.WithLevel(log.LevelDebug)))

	log.Debug("welcome to xbase-framework")
	log.Info("welcome to xbase-framework")
	log.Warn("welcome to xbase-framework")
	log.Error("welcome to xbase-framework")
}
