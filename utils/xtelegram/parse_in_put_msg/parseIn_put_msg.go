package parseInPutMsg

import "strings"

func ParseInPutMsg(msg string) []string {
	if len(msg) == 0 {
		return nil
	}
	//用换符号切割
	msg = strings.Replace(msg, "\r\n", " ", -1)
	msg = strings.Replace(msg, "\n", " ", -1)
	//英文，
	msg = strings.Replace(msg, ",", " ", -1)
	//英文;
	msg = strings.Replace(msg, ";", " ", -1)
	// |
	msg = strings.Replace(msg, "|", " ", -1)

	return strings.Split(msg, " ")
}
