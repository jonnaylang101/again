package again

import "net/http"

type ClientOptions struct {
	Transport  http.RoundTripper
	Whitelist  []int
	MaxRetries int
}

type ClientOption func(*ClientOptions)

func NewClient(maxRetries int, options ...ClientOption) *http.Client {
	ops := &ClientOptions{
		Transport:  http.DefaultTransport,
		Whitelist:  DefaultWhitelist,
		MaxRetries: maxRetries,
	}

	for _, opt := range options {
		opt(ops)
	}

	t := &retryTransport{
		transport:  ops.Transport,
		whitelist:  ops.Whitelist,
		maxRetries: ops.MaxRetries,
	}

	return &http.Client{
		Transport: t,
	}
}
