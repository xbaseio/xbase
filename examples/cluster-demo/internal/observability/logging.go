package observability

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xbaseio/xbase/log"
)

func Info(event string, kv ...any) {
	log.Infof("%s", render(event, kv...))
}

func Warn(event string, err error, kv ...any) {
	items := kv
	if err != nil {
		items = append(items, "err", err)
	}
	log.Warnf("%s", render(event, items...))
}

func Error(event string, err error, kv ...any) {
	items := kv
	if err != nil {
		items = append(items, "err", err)
	}
	log.Errorf("%s", render(event, items...))
}

func render(event string, kv ...any) string {
	fields := make(map[string]string, len(kv)/2+1)
	fields["event"] = event

	for i := 0; i+1 < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		fields[key] = stringify(kv[i+1])
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, quote(fields[key])))
	}

	return strings.Join(parts, " ")
}

func stringify(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case fmt.Stringer:
		return val.String()
	case error:
		return val.Error()
	default:
		return fmt.Sprint(v)
	}
}

func quote(v string) string {
	if v == "" {
		return `""`
	}
	if strings.ContainsAny(v, " \t\r\n\"") {
		return fmt.Sprintf("%q", v)
	}
	return v
}
