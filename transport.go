package again

import (
	"bytes"
	"errors"
	"fmt"
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
	transport  http.RoundTripper
	whitelist  []int
	maxRetries int
	notifyFunc backoff.Notify
	response   *http.Response
	retryCount int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := cacheRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("[again.RoundTrip] failed to cache request body: %w", err)
	}

	if err = backoff.RetryNotify(t.try(req, body), t.initBackoff(), t.notifyFunc); err != nil {
		return nil, err
	}

	return t.response, nil
}

var (
	fmtErrNonRetryable = "[again.try] non-retryable error: %v"
	fmtErrRetryable    = "[again.try] retry error: %v"
	fmtErrFinal        = "[again.try] final error: %v"
)

func (t *retryTransport) try(req *http.Request, bodyData []byte) backoff.Operation {
	return func() error {
		var err error
		req.Body = io.NopCloser(bytes.NewBuffer(bodyData))
		res, err := t.transport.RoundTrip(req)
		if err != nil {
			return backoff.Permanent(fmt.Errorf(fmtErrNonRetryable, err))
		}

		t.response = res
		if res.StatusCode < http.StatusBadRequest {
			return nil
		}

		if t.retryCount++; t.retryCount > t.maxRetries {
			return backoff.Permanent(fmt.Errorf(fmtErrFinal, errors.New(http.StatusText(res.StatusCode))))
		}

		if tryAgain(res.StatusCode, t.whitelist) {
			if err = flushResponseBody(res); err != nil {
				return backoff.Permanent(fmt.Errorf(fmtErrNonRetryable, err))
			}
			return fmt.Errorf(fmtErrRetryable, errors.New(http.StatusText(res.StatusCode)))
		}

		return backoff.Permanent(fmt.Errorf(fmtErrNonRetryable, err))
	}
}

func (t *retryTransport) initBackoff() backoff.BackOff {
	return backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(t.maxRetries))
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

func tryAgain(statusCode int, whitelist []int) bool {
	for _, code := range whitelist {
		if statusCode == code {
			return true
		}
	}
	return false
}

// flush the request body to allow another retry
func flushResponseBody(res *http.Response) error {
	if res.Body == nil {
		return nil
	}

	_, err := io.Copy(io.Discard, res.Body)

	return err
}
