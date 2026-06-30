package tgwebhook

import (
	"github.com/xbaseio/xbase/utils/xhttp"
)

type client struct {
	client *xhttp.Client
}

func NewClient(token string) *client {
	baseUrl := "https://api.telegram.org/bot" + token
	c := &client{client: xhttp.NewClient()}
	c.client.SetBaseUrl(baseUrl)
	c.client.SetHeaders(map[string]string{
		"Accept": "/",
	})
	return c
}

// Get 执行Get请求
func (c *client) Get(url string, req, resp any) error {
	return c.request(xhttp.MethodGet, url, req, resp)
}

// 执行请求
func (c *client) request(method string, url string, req, resp any) error {
	res, err := c.client.Request(method, url, req)
	if err != nil {
		return err
	}

	return res.ScanBody(resp)
}
