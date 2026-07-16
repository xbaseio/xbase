package file

import (
	"reflect"
	"time"

	"github.com/xbaseio/xbase/etc"
)

const (
	defaultPath              = "./log/xbase.log"
	defaultMaxAge            = "7d"
	defaultMaxSize           = "500M"
	defaultRotate            = RotateNone
	defaultCompress          = false
	defaultClassifiedStorage = false
	defaultFormat            = FormatText
)

const (
	defaultPathKey                    = "etc.log.file.path"
	defaultFormatKey                  = "etc.log.file.format"
	defaultMaxAgeKey                  = "etc.log.file.maxAge"
	defaultMaxSizeKey                 = "etc.log.file.maxSize"
	defaultRotateKey                  = "etc.log.file.rotate"
	defaultCompressKey                = "etc.log.file.compress"
	defaultNestedClassifiedStorageKey = "etc.log.file.classifiedStorage"
	defaultFlatFormatKey              = "etc.log.format"
	defaultFlatMaxAgeKey              = "etc.log.fileMaxAge"
	defaultFlatMaxSizeKey             = "etc.log.fileMaxSize"
	defaultFlatRotateKey              = "etc.log.fileCutRule"
	defaultClassifiedStorageKey       = "etc.log.classifiedStorage"
)

type Option func(o *options)

type options struct {
	path              string        // 文件路径
	format            Format        // 输出格式
	maxAge            time.Duration // 文件最大留存时间
	maxSize           int64         // 单个文件最大尺寸
	rotate            Rotate        // 文件反转规则
	compress          bool          // 是否对轮换的日志文件进行压缩
	classifiedStorage bool          // 是否按日志级别分别存储
}

func defaultOptions() *options {
	o := &options{
		path:              etc.Get(defaultPathKey, defaultPath).String(),
		format:            Format(etc.Get(defaultFormatKey, defaultFormat).String()),
		maxAge:            etc.Get(defaultMaxAgeKey, defaultMaxAge).Duration(),
		maxSize:           int64(etc.Get(defaultMaxSizeKey, defaultMaxSize).B()),
		rotate:            Rotate(etc.Get(defaultRotateKey, defaultRotate).String()),
		compress:          etc.Get(defaultCompressKey, defaultCompress).Bool(),
		classifiedStorage: etc.Get(defaultClassifiedStorageKey, defaultClassifiedStorage).Bool(),
	}

	// The legacy Due configuration keeps all file settings directly under [log].
	if value := etc.Get("etc.log.file"); value.Kind() == reflect.String {
		o.path = value.String()
	}
	if value := etc.Get(defaultFlatFormatKey); value.Kind() == reflect.String {
		o.format = Format(value.String())
	}
	if etc.Has(defaultFlatMaxAgeKey) {
		o.maxAge = etc.Get(defaultFlatMaxAgeKey).Duration()
	}
	if etc.Has(defaultFlatMaxSizeKey) {
		o.maxSize = int64(etc.Get(defaultFlatMaxSizeKey).Int()) * 1024 * 1024
	}
	if etc.Has(defaultFlatRotateKey) {
		o.rotate = Rotate(etc.Get(defaultFlatRotateKey).String())
	}
	if etc.Has(defaultNestedClassifiedStorageKey) {
		o.classifiedStorage = etc.Get(defaultNestedClassifiedStorageKey).Bool()
	}
	return o
}

// WithPath 设置文件路径
func WithPath(path string) Option {
	return func(o *options) { o.path = path }
}

// WithFormat 设置输出格式
func WithFormat(format Format) Option {
	return func(o *options) { o.format = format }
}

// WithMaxAge 设置文件最大留存时间
func WithMaxAge(maxAge time.Duration) Option {
	return func(o *options) { o.maxAge = maxAge }
}

// WithMaxSize 设置单个文件最大尺寸
func WithMaxSize(maxSize int64) Option {
	return func(o *options) { o.maxSize = maxSize }
}

// WithRotate 设置文件反转规则
func WithRotate(rotate Rotate) Option {
	return func(o *options) { o.rotate = rotate }
}

// WithCompress 设置是否对轮换日志文件进行压缩
func WithCompress(compress bool) Option {
	return func(o *options) { o.compress = compress }
}

// WithClassifiedStorage sets whether logs are stored in files by level.
func WithClassifiedStorage(enable bool) Option {
	return func(o *options) { o.classifiedStorage = enable }
}
