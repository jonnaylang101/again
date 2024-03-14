package again

import (
	"bytes"
	"io"
	"net/http"

	"github.com/cenkalti/backoff"
)

var DefaultWhitelist = []int{
	http.StatusTooManyRequests,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
	http.StatusInternalServerError,
	http.StatusBadGateway,
}

type retryTransport struct {
	transport http.RoundTripper
	whitelist []int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	byt, err := cacheRequestBody(req)
	if err != nil {
		return nil, err
	}

	operation := func() error {
		req.Body = io.NopCloser(bytes.NewBuffer(byt))
		res, err := t.transport.RoundTrip(req)
		if err != nil {
			return err
		}

		for _, code := range t.whitelist {
			if res.StatusCode == code {
				flushResponseBody(res)
				return err
			}
		}

		return backoff.Permanent(err)
	}

	if err = backoff.Retry(operation, backoff.NewExponentialBackOff()); err != nil {
		return nil, err
	}

	var res *http.Response
	return res, nil
}

type ClientOptions struct {
	Transport http.RoundTripper
}

func NewClient() *http.Client {
	t := &retryTransport{
		transport: http.DefaultTransport,
		whitelist: DefaultWhitelist,
	}

	return &http.Client{
		Transport: t,
	}
}

// cache the request body in a new buffer for reuse in each retry
func cacheRequestBody(req *http.Request) (byt []byte, err error) {
	byt = make([]byte, 0)

	if req.Body == nil {
		return byt, nil
	}

	byt, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	return byt, nil
}

// flush the request body to allow another retry
func flushResponseBody(res *http.Response) error {
	if res.Body != nil {
		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			return err
		}
	}

	return nil
}
