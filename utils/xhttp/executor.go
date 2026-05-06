package xhttp

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Request interface {
	Next() (*Response, error)
	Request() *http.Request
}

type executor struct {
	client  *Client
	request *http.Request
}

func (e *executor) Next() (*Response, error) {
	if v := e.request.Context().Value(middlewareKey); v != nil {
		if m, ok := v.(*middleware); ok {
			return m.Next()
		}
	}

	return e.doRequest()
}

func (e *executor) Request() *http.Request {
	return e.request
}

func (e *executor) call(req *http.Request) (resp *Response, err error) {
	e.request = req
	if middlewares := e.client.getMiddlewares(); len(middlewares) > 0 {
		handlers := make([]MiddlewareFunc, 0, len(middlewares)+1)
		handlers = append(handlers, e.client.getMiddlewares()...)
		handlers = append(handlers, func(r Request) (*Response, error) {
			return e.doRequest()
		})
		e.request = e.request.WithContext(context.WithValue(e.request.Context(), middlewareKey, &middleware{
			req:      e,
			handlers: handlers,
			index:    -1,
		}))
		resp, err = e.Next()
	} else {
		resp, err = e.doRequest()
	}

	return
}

// nitiate an HTTP request and return the response data.
func (e *executor) doRequest() (resp *Response, err error) {
	resp = &Response{
		Request: e.request,
	}

	retryCount := e.client.retryCount
	retryInterval := e.client.retryInterval
	ctx := e.client.ctx

	for attempt := 0; ; attempt++ {
		// 重试时，如果是 POST/PUT 等带 Body 的请求，需要重置 Body
		// 否则第二次 Do 可能会出现空 body / ContentLength 不匹配
		if attempt > 0 && e.request.Body != nil {
			if e.request.GetBody == nil {
				// Body 不可重复读取，不能安全重试
				return nil, err
			}

			body, bodyErr := e.request.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			e.request.Body = body
		}

		resp.Response, err = e.client.Do(e.request)
		if err == nil {
			return resp, nil
		}

		// 有些情况下 err != nil 也可能返回 Response，防御性关闭
		if resp.Response != nil && resp.Response.Body != nil {
			_ = resp.Response.Body.Close()
			resp.Response = nil
		}

		// 已经达到最大重试次数
		if attempt >= retryCount {
			return nil, err
		}

		// 如果请求已经被取消，直接返回
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// 等待重试间隔，但要支持 ctx cancel
		if retryInterval > 0 {
			timer := time.NewTimer(retryInterval)

			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return nil, ctx.Err()
			}
		}
	}
}

func (e *executor) makeUrl(url string) string {
	if e.client.baseUrl == "" {
		return url
	}

	matched, err := regexp.MatchString(`(?i)^(http|https)://[-a-zA-Z0-9+&@#/%?=~_|,!:.;]*`, url)
	if err == nil && matched {
		return url
	}

	return strings.TrimRight(e.client.baseUrl, "/") + "/" + strings.TrimLeft(url, "/")
}
