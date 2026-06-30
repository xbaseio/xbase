package expand

import "os"

func Expand(src string, replaces map[string]string) string {
	if len(replaces) == 0 {
		return src
	}

	body := os.Expand(src, func(s string) string {
		return replaces[s]
	})
	return body
}
