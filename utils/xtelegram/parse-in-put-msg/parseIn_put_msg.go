package parseInPutMsg

import "strings"

func ParseInPutMsg(msg string) []string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return nil
	}

	// 统一把各种分隔符替换成空格
	replacer := strings.NewReplacer(
		"\r\n", " ",
		"\n", " ",
		"\r", " ",

		",", " ",
		";", " ",
		"|", " ",

		"，", " ",
		"；", " ",
		"｜", " ",
		"、", " ",
		"\t", " ",
	)

	msg = replacer.Replace(msg)

	// Fields 会自动去掉连续空格和空字符串
	return strings.Fields(msg)
}
