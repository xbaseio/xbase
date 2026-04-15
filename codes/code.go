package codes

import (
	"errors"
	"fmt"
	"io"
)

var (
	OK               = NewCode(0, "ok")
	Canceled         = NewCode(1, "canceled")
	Unknown          = NewCode(2, "unknown")
	InvalidArgument  = NewCode(3, "invalid argument")
	DeadlineExceeded = NewCode(4, "deadline exceeded")
	NotFound         = NewCode(5, "not found")
	InternalError    = NewCode(6, "internal error")
	Unauthorized     = NewCode(7, "unauthorized")
	IllegalInvoke    = NewCode(8, "illegal invoke")
	IllegalRequest   = NewCode(9, "illegal request")
	TooManyRequests  = NewCode(10, "too many requests")
)

type Code struct {
	code    int
	message string
}

func NewCode(code int, message ...string) *Code {
	c := &Code{code: code}
	if len(message) > 0 {
		c.message = message[0]
	}
	return c
}

func (c *Code) Code() int {
	if c == nil {
		return Unknown.code
	}
	return c.code
}

func (c *Code) Message() string {
	if c == nil {
		return Unknown.message
	}
	return c.message
}

func (c *Code) WithCode(code int) *Code {
	if c == nil {
		return &Code{code: code}
	}
	return &Code{
		code:    code,
		message: c.message,
	}
}

func (c *Code) WithMessage(message string) *Code {
	if c == nil {
		return &Code{message: message}
	}
	return &Code{
		code:    c.code,
		message: message,
	}
}

func (c *Code) WithMessagef(format string, a ...any) *Code {
	return c.WithMessage(fmt.Sprintf(format, a...))
}

func (c *Code) String() string {
	if c == nil {
		return "code error: code = 2 desc = unknown"
	}
	if c.message == "" {
		return fmt.Sprintf("code error: code = %d", c.code)
	}
	return fmt.Sprintf("code error: code = %d desc = %s", c.code, c.message)
}

func (c *Code) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		if c == nil {
			_, _ = io.WriteString(s, "2:unknown")
			return
		}
		if c.message != "" {
			_, _ = io.WriteString(s, fmt.Sprintf("%d:%s", c.code, c.message))
		} else {
			_, _ = io.WriteString(s, fmt.Sprintf("%d", c.code))
		}
	case 'd':
		if c == nil {
			_, _ = io.WriteString(s, "2")
			return
		}
		_, _ = io.WriteString(s, fmt.Sprintf("%d", c.code))
	case 'v':
		_, _ = io.WriteString(s, c.String())
	default:
		_, _ = io.WriteString(s, c.String())
	}
}

func (c *Code) Err() error {
	if c == nil || c.code == OK.code {
		return nil
	}
	return &codeError{code: c}
}

type codeError struct {
	code *Code
}

func (e *codeError) Error() string {
	if e == nil || e.code == nil {
		return Unknown.String()
	}
	return e.code.String()
}

func (e *codeError) Code() *Code {
	if e == nil || e.code == nil {
		return Unknown
	}
	return e.code
}

func Convert(err error) *Code {
	if err == nil {
		return OK
	}

	type coder interface {
		Code() *Code
	}

	var c coder
	if errors.As(err, &c) {
		if code := c.Code(); code != nil {
			return code
		}
	}

	return Unknown
}
