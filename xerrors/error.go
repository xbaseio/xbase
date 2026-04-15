package xerrors

import (
	stderrors "errors"
	"fmt"
	"io"

	"github.com/xbaseio/xbase/codes"
	"github.com/xbaseio/xbase/core/stack"
)

type Error struct {
	err   error
	text  string
	code  *codes.Code
	stack *stack.Stack
}

// New 仅文本错误
func New(text string) *Error {
	return &Error{text: text}
}

// NewCode 带错误码
func NewCode(code *codes.Code, text ...string) *Error {
	e := &Error{code: code}
	if len(text) > 0 {
		e.text = text[0]
	}
	return e
}

// Wrap 包装原始错误
func Wrap(err error, args ...any) *Error {
	if err == nil && len(args) == 0 {
		return nil
	}

	e := &Error{err: err}
	fillError(e, args...)
	return e
}

// WithStack 包装并附带堆栈
func WithStack(err error, args ...any) *Error {
	if err == nil && len(args) == 0 {
		return nil
	}

	e := &Error{
		err:   err,
		stack: stack.Callers(1, stack.Full),
	}
	fillError(e, args...)
	return e
}

func fillError(e *Error, args ...any) {
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			e.text = v
		case error:
			e.err = v
		case *codes.Code:
			e.code = v
		}
	}
}

// Error 实现 error
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	parts := make([]string, 0, 3)

	if e.code != nil && e.code != codes.OK {
		parts = append(parts, e.code.String())
	}
	if e.text != "" {
		parts = append(parts, e.text)
	}
	if e.err != nil && e.err.Error() != "" {
		parts = append(parts, e.err.Error())
	}

	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return join(parts, ": ")
	}
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	for i := 1; i < len(parts); i++ {
		s += sep + parts[i]
	}
	return s
}

// Code 返回错误码
func (e *Error) Code() *codes.Code {
	if e == nil || e.code == nil {
		return nil
	}
	return e.code
}

// Unwrap 支持 errors.As / errors.Is / errors.Unwrap
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Next 兼容你原来的接口
func (e *Error) Next() error {
	return e.Unwrap()
}

// Stack 返回堆栈
func (e *Error) Stack() *stack.Stack {
	if e == nil {
		return nil
	}
	return e.stack
}

// Cause 返回最底层错误
func (e *Error) Cause() error {
	if e == nil {
		return nil
	}

	cur := error(e)
	for cur != nil {
		next := stderrors.Unwrap(cur)
		if next == nil {
			return cur
		}
		if next == cur {
			return cur
		}
		cur = next
	}
	return nil
}

// Replace 替换文本
func (e *Error) Replace(text string, condition ...*codes.Code) error {
	if e == nil {
		return nil
	}

	if len(condition) == 0 {
		e.text = text
		return e
	}

	for _, c := range condition {
		if c == e.code {
			e.text = text
			break
		}
	}

	return e
}

// String 格式化错误信息
func (e *Error) String() string {
	return fmt.Sprintf("%+v", e)
}

func (e *Error) localText() string {
	if e == nil {
		return ""
	}

	if e.text != "" {
		return e.text
	}
	if e.code != nil && e.code != codes.OK {
		return e.code.String()
	}
	return ""
}

// Format
// %s  : 本级错误信息
// %v  : 全链错误信息
// %+v : 全链错误信息 + 堆栈
func (e *Error) Format(s fmt.State, verb rune) {
	if e == nil {
		return
	}

	switch verb {
	case 's':
		txt := e.localText()
		if txt == "" {
			txt = e.Error()
		}
		_, _ = io.WriteString(s, txt)

	case 'v':
		if !s.Flag('+') {
			_, _ = io.WriteString(s, e.Error())
			return
		}

		_, _ = io.WriteString(s, e.Error())

		var (
			i         = 1
			cur error = e
		)

		for cur != nil {
			if i == 1 {
				_, _ = io.WriteString(s, "\nDetails:\n")
			}

			if ce, ok := cur.(*Error); ok {
				_, _ = fmt.Fprintf(s, "%d. %s\n", i, ce.localText())
				if ce.stack != nil {
					_, _ = io.WriteString(s, "   Stack:\n")
					for j, f := range ce.stack.Frames() {
						_, _ = fmt.Fprintf(s, "   %d) %s\n      %s:%d\n", j+1, f.Function, f.File, f.Line)
					}
				}
				cur = ce.Unwrap()
			} else {
				_, _ = fmt.Fprintf(s, "%d. %s\n", i, cur.Error())
				cur = stderrors.Unwrap(cur)
			}

			i++
		}

	default:
		_, _ = io.WriteString(s, e.Error())
	}
}

// Code 从错误链提取 code
func Code(err error) *codes.Code {
	if err == nil {
		return nil
	}

	type coder interface {
		Code() *codes.Code
	}

	var c coder
	if stderrors.As(err, &c) {
		return c.Code()
	}

	return nil
}

// Next 返回下一个错误
func Next(err error) error {
	if err == nil {
		return nil
	}
	return stderrors.Unwrap(err)
}

// Cause 返回根因错误
func Cause(err error) error {
	if err == nil {
		return nil
	}

	cur := err
	for cur != nil {
		next := stderrors.Unwrap(cur)
		if next == nil || next == cur {
			return cur
		}
		cur = next
	}
	return cur
}

// Stack 返回堆栈
func Stack(err error) *stack.Stack {
	if err == nil {
		return nil
	}

	type stacker interface {
		Stack() *stack.Stack
	}

	var s stacker
	if stderrors.As(err, &s) {
		return s.Stack()
	}

	return nil
}

// Replace 替换文本
func Replace(err error, text string, condition ...*codes.Code) error {
	if err == nil {
		return nil
	}

	type replacer interface {
		Replace(text string, condition ...*codes.Code) error
	}

	var r replacer
	if stderrors.As(err, &r) {
		return r.Replace(text, condition...)
	}

	return err
}

// Is 包装标准库 errors.Is
func Is(err, target error) bool { return stderrors.Is(err, target) }

// As 包装标准库 errors.As
func As(err error, target any) bool { return stderrors.As(err, target) }

// Unwrap 包装标准库 errors.Unwrap
func Unwrap(err error) error { return stderrors.Unwrap(err) }
