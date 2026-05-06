package xcasbin

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm/logger"
)

// 使用 SyncedEnforcer，支持 StartAutoLoadPolicy / StopAutoLoadPolicy
type Enforcer = casbin.SyncedEnforcer
type Logger = logger.Interface

const (
	defaultAutoloadDuration = time.Minute
)

var (
	ErrNilOptions  = errors.New("xcasbin: options is nil")
	ErrEmptyModel  = errors.New("xcasbin: model is empty")
	ErrNilDatabase = errors.New("xcasbin: database is nil")
)

type Options struct {
	Model    string        `json:"model"`    // model config file path
	Debug    bool          `json:"debug"`    // debug mode
	Enable   bool          `json:"enable"`   // enable permission, false means allow all
	Autoload bool          `json:"autoload"` // auto load policy
	Duration time.Duration `json:"duration"` // auto load duration
	Database interface{}   `json:"database"` // database instance
	Table    string        `json:"table"`    // database policy table name
	Logger   Logger        `json:"logger"`   // database logger interface
}

func NewEnforcer(opts *Options) (*Enforcer, error) {
	cfg, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}

	adp, err := newAdapter(cfg.Database, cfg.Table, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("xcasbin: new adapter failed: %w", err)
	}

	enforcer, err := casbin.NewSyncedEnforcer(cfg.Model, adp)
	if err != nil {
		return nil, fmt.Errorf("xcasbin: new synced enforcer failed: %w", err)
	}

	// 是否开启 casbin 日志
	enforcer.EnableLog(cfg.Debug)

	// 是否开启权限校验
	// 注意：Enable=false 时，Enforce 会直接放行
	enforcer.EnableEnforce(cfg.Enable)

	// AddPolicy / RemovePolicy / UpdatePolicy 时自动保存到 DB
	enforcer.EnableAutoSave(true)

	// 定时从 DB 加载最新策略
	if cfg.Autoload {
		enforcer.StartAutoLoadPolicy(cfg.Duration)
	}

	return enforcer, nil
}

func normalizeOptions(opts *Options) (Options, error) {
	if opts == nil {
		return Options{}, ErrNilOptions
	}

	cfg := *opts

	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		return Options{}, ErrEmptyModel
	}

	if cfg.Database == nil {
		return Options{}, ErrNilDatabase
	}

	cfg.Table = strings.TrimSpace(cfg.Table)

	if cfg.Autoload && cfg.Duration <= 0 {
		cfg.Duration = defaultAutoloadDuration
	}

	return cfg, nil
}
