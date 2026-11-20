package etc

import (
	"encoding/json"
	"os"
	"strings"
	"unicode"

	"github.com/xbaseio/xbase/config"
	"github.com/xbaseio/xbase/config/file/core"
	"github.com/xbaseio/xbase/core/value"
	"github.com/xbaseio/xbase/env"
	"github.com/xbaseio/xbase/flag"
)

// etc主要被当做项目启动配置存在；常用于集群配置、服务组件配置等。
// 配置默认读取文件；叶子配置项可通过 XBASE_CONFIG_<配置路径> 环境变量覆盖。
// etc无法通过master管理服进行修改。
// 如想在业务使用配置，推荐使用config配置中心进行实现。
// config配置中心的配置信息可通过master管理服进行动态修改。

const (
	dueEtcEnvName  = "XBASE_ETC"
	dueEtcArgName  = "etc"
	defaultEtcPath = "./etc"
)

var globalConfigurator config.Configurator

func init() {
	path := env.Get(dueEtcEnvName, defaultEtcPath).String()
	path = flag.String(dueEtcArgName, path)
	globalConfigurator = config.NewConfigurator(config.WithSources(core.NewSource(path, config.ReadOnly)))
}

// SetConfigurator 设置配置器
func SetConfigurator(configurator config.Configurator) {
	if globalConfigurator != nil {
		globalConfigurator.Close()
	}

	globalConfigurator = configurator
}

// GetConfigurator 获取配置器
func GetConfigurator() config.Configurator {
	return globalConfigurator
}

// Has 是否存在配置
func Has(pattern string) bool {
	return globalConfigurator.Has(pattern)
}

// Get 获取配置值
func Get(pattern string, def ...any) value.Value {
	return overrideWithEnvironment(pattern, globalConfigurator.Get(pattern, def...))
}

func overrideWithEnvironment(pattern string, configured value.Value) value.Value {
	if raw, ok := os.LookupEnv(configEnvName(pattern)); ok {
		if configured.IsArray() || configured.IsSlice() {
			var decoded []any
			if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
				return value.NewValue(decoded)
			}
			return configured
		}
		return value.NewValue(raw)
	}
	return configured
}

func configEnvName(pattern string) string {
	pattern = strings.TrimPrefix(pattern, "etc.")
	var name strings.Builder
	name.WriteString("XBASE_CONFIG_")
	previousLower := false
	for _, r := range pattern {
		if r == '.' || r == '-' {
			name.WriteByte('_')
			previousLower = false
			continue
		}
		if unicode.IsUpper(r) && previousLower {
			name.WriteByte('_')
		}
		name.WriteRune(unicode.ToUpper(r))
		previousLower = unicode.IsLower(r) || unicode.IsDigit(r)
	}
	return name.String()
}

// Set 设置配置值
func Set(pattern string, value any) error {
	return globalConfigurator.Set(pattern, value)
}

// Match 匹配多个规则
func Match(patterns ...string) config.Matcher {
	return globalConfigurator.Match(patterns...)
}

// Close 关闭配置监听
func Close() {
	globalConfigurator.Close()
}
