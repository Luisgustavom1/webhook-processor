package http

import (
	"io"
	"net/http"
	"time"
)

type HTTPClient struct {
	Client *http.Client
}

type ClientOpts struct {
	Timeout time.Duration
}

func NewClient(opts ClientOpts) *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Timeout: opts.Timeout,
		},
	}
}

func (c *HTTPClient) Get(url string) (*http.Response, error) {
	return c.Client.Get(url)
}

func (c *HTTPClient) Post(url string, bodyType string, body io.Reader) (*http.Response, error) {
	return c.Client.Post(url, bodyType, body)
}
