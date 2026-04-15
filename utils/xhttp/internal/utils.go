package internal

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/xbaseio/xbase/utils/xconv"
)

const fileUploadingKey = "@file:"

func BuildParams(params any) string {
	switch v := params.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case []any:
		if len(v) > 0 {
			params = v[0]
		} else {
			params = nil
		}
	}

	m := make(map[string]any)

	if params != nil {
		if b, err := json.Marshal(params); err != nil {
			return xconv.String(params)
		} else if err = json.Unmarshal(b, &m); err != nil {
			return xconv.String(params)
		}
	} else {
		return ""
	}

	urlEncode := true

	if len(m) == 0 {
		return xconv.String(params)
	}

	for k, v := range m {
		if strings.Contains(k, fileUploadingKey) || strings.Contains(xconv.String(v), fileUploadingKey) {
			urlEncode = false
			break
		}
	}

	var (
		s   = ""
		str = ""
	)

	for k, v := range m {
		if len(str) > 0 {
			str += "&"
		}
		s = xconv.String(v)
		if urlEncode && len(s) > len(fileUploadingKey) && strings.Compare(s[0:len(fileUploadingKey)], fileUploadingKey) != 0 {
			s = url.QueryEscape(s)
		}
		str += k + "=" + s
	}

	return str
}
