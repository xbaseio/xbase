package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// WebAppInfo 描述一个 Web 应用。
type WebAppInfo struct {
	URL string `json:"url"` // 要打开的 Web 应用的 HTTPS URL，附加数据根据初始化 Web 应用的说明进行处理
}

func (j WebAppInfo) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *WebAppInfo) Scan(value any) error {
	if value == nil {
		j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid Scan Source")
	}
	return json.Unmarshal(s, j)
}
func (j WebAppInfo) IsNull() bool {
	return j.URL == ""
}
func (j WebAppInfo) Clone() *WebAppInfo {
	if j.IsNull() {
		return nil
	}
	return &WebAppInfo{
		URL: j.URL, // 将在 UsersShared 对象中接收到的请求的有符号 32 位标识符。必须在消息中唯一。

	}
}
