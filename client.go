package again

import (
	"net/http"

	"github.com/cenkalti/backoff"
)

type ClientOptions struct {
	Transport  http.RoundTripper
	Whitelist  []int
	MaxRetries int
	NotifyFunc backoff.Notify
}

type ClientOption func(*ClientOptions)

func NewClient(maxRetries int, options ...ClientOption) *http.Client {
	ops := &ClientOptions{
		Transport:  http.DefaultTransport,
		Whitelist:  DefaultWhitelist,
		MaxRetries: maxRetries,
		NotifyFunc: nil,
	}

	for _, opt := range options {
		opt(ops)
	}

	t := &retryTransport{
		transport:  ops.Transport,
		whitelist:  ops.Whitelist,
		maxRetries: ops.MaxRetries,
		notifyFunc: ops.NotifyFunc,
	}

	return &http.Client{
		Transport: t,
	}
}
