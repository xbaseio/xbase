package sse

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Event struct {
	ID    string
	Event string
	Retry int
	Data  any
}

func (e Event) Bytes() []byte {
	var builder strings.Builder

	if e.ID != "" {
		builder.WriteString("id: ")
		builder.WriteString(e.ID)
		builder.WriteString("\n")
	}
	if e.Event != "" {
		builder.WriteString("event: ")
		builder.WriteString(e.Event)
		builder.WriteString("\n")
	}
	if e.Retry > 0 {
		builder.WriteString(fmt.Sprintf("retry: %d\n", e.Retry))
	}

	for _, line := range splitDataLines(e.Data) {
		builder.WriteString("data: ")
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	builder.WriteString("\n")
	return []byte(builder.String())
}

func splitDataLines(data any) []string {
	if data == nil {
		return []string{""}
	}

	var raw string
	switch v := data.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		buf, err := json.Marshal(v)
		if err != nil {
			raw = fmt.Sprint(v)
		} else {
			raw = string(buf)
		}
	}

	return strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
}
