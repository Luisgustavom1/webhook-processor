package http

import (
	"io"
	"net/http"
	"time"
)

type Response = http.Response

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
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *HTTPClient) Post(url string, bodyType string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.Client.Do(req)
}
