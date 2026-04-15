package xhttp

type MiddlewareFunc = func(r Request) (*Response, error)

const middlewareKey = "__httpClientMiddlewareKey"

type middleware struct {
	err      error
	req      Request
	resp     *Response
	index    int
	handlers []MiddlewareFunc
}

// Next exec the next middleware.
func (m *middleware) Next() (*Response, error) {
	m.index++

	if m.index < len(m.handlers) {
		if m.resp, m.err = m.handlers[m.index](m.req); m.err != nil {
			return m.resp, m.err
		}
	}

	return m.resp, m.err
}
