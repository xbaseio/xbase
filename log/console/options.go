package console

import (
	"reflect"

	"github.com/xbaseio/xbase/etc"
)

const (
	defaultFormat = FormatText
)

const (
	defaultFormatKey = "etc.log.console.format"
)

type Option func(o *options)

type options struct {
	format Format // 输出格式
}

func defaultOptions() *options {
	format := etc.Get(defaultFormatKey, defaultFormat).String()
	if value := etc.Get("etc.log.format"); value.Kind() == reflect.String {
		format = value.String()
	}
	return &options{
		format: Format(format),
	}
}

// WithFormat 设置输出格式
func WithFormat(format Format) Option {
	return func(o *options) { o.format = format }
}
