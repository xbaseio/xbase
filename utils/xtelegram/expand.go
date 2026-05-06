package xtelegram

import "os"

func Expand(text string, replaces map[string]string) string {
	if len(replaces) == 0 {
		return text
	}

	return os.Expand(text, func(key string) string {
		return replaces[key]
	})
}
