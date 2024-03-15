package again

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/cenkalti/backoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_retryTransport_RoundTrip(t *testing.T) {
	t.Run("when the body can't be copied we should return a 'file already close' error", func(t *testing.T) {
		// create a new file and close it to simulate a closed file
		f, err := os.Create("file")
		require.NoError(t, err)
		f.Close()

		t.Cleanup(func() {
			os.Remove("file")
		})

		req := &http.Request{
			Body: io.NopCloser(f),
		}

		retryTransport := &retryTransport{
			transport: http.DefaultTransport,
		}

		_, err = retryTransport.RoundTrip(req)

		assert.ErrorContains(t, err, "file already closed")
	})
}

type stubRoundTripper struct {
	res *http.Response
	err error
}

func (srt *stubRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return srt.res, srt.err
}

func Test_cacheRequestBody(t *testing.T) {
	t.Run("when the body has nil value, we should not error", func(t *testing.T) {
		req := &http.Request{}
		byt, err := cacheRequestBody(req)

		assert.NoError(t, err)
		assert.Equal(t, []byte{}, byt)
	})

	t.Run("when the body holds data, we should cache that data", func(t *testing.T) {
		expect := []byte("hello")
		req := &http.Request{
			Body:          io.NopCloser(bytes.NewBuffer(expect)),
			ContentLength: int64(len(expect)),
		}

		byt, err := cacheRequestBody(req)

		assert.NoError(t, err)
		assert.Equal(t, []byte("hello"), byt)
	})
}

func Test_flushResponseBody(t *testing.T) {
	t.Run("when the response body is nil, we should not error", func(t *testing.T) {
		res := &http.Response{}
		err := flushResponseBody(res)

		assert.NoError(t, err)
	})

	t.Run("when the response body holds data, we should flush that data", func(t *testing.T) {
		res := &http.Response{
			Body: io.NopCloser(bytes.NewBuffer([]byte("hello"))),
		}
		err := flushResponseBody(res)

		assert.NoError(t, err)
		actual, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte{}, actual)
	})
}

func Test_retryTransport_try(t *testing.T) {
	t.Run("when the RoundTripper setup errors, we should return a non-retryable error", func(t *testing.T) {
		whitelist := []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
		}

		expect := errors.New("some error message")

		transport := &retryTransport{
			transport: &stubRoundTripper{
				res: nil,
				err: expect,
			},
			whitelist: whitelist,
		}

		err := transport.try(&http.Request{}, []byte{})()
		assert.ErrorContains(t, err, expect.Error())
	})

	t.Run("when the status code is in the whitelist, we should return a retryable error", func(t *testing.T) {
		whitelist := []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
		}

		transport := &retryTransport{
			transport: &stubRoundTripper{
				res: &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(bytes.NewBuffer([]byte("res body data"))),
				},
				err: nil,
			},
			whitelist: whitelist,
		}

		err := transport.try(&http.Request{}, []byte("body data"))()

		assert.ErrorContains(t, err, "retryable error")
		assert.False(t, errors.Is(err, (&backoff.PermanentError{})))
	})

	t.Run("when the status code is not in the whitelist, we should return a non-retryable error", func(t *testing.T) {
		whitelist := []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
		}

		transport := &retryTransport{
			transport: &stubRoundTripper{
				res: &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBuffer([]byte("res body data"))),
				},
				err: nil,
			},
			whitelist: whitelist,
		}

		op := transport.try(&http.Request{}, []byte{})

		err := op()
		assert.ErrorContains(t, err, "non-retryable error")
		assert.IsType(t, &backoff.PermanentError{}, err)
	})

	t.Run("when the response body is currupt, we should return a non-retyable error", func(t *testing.T) {
		whitelist := []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
		}

		f, err := os.Create("file")
		require.NoError(t, err)
		f.Close()

		t.Cleanup(func() {
			os.Remove("file")
		})

		transport := &retryTransport{
			transport: &stubRoundTripper{
				res: &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(f),
				},
				err: nil,
			},
			whitelist: whitelist,
		}

		err = transport.try(&http.Request{}, []byte{})()

		assert.ErrorContains(t, err, "non-retryable error")
		assert.IsType(t, &backoff.PermanentError{}, err)
	})

}

func Test_tryAgain(t *testing.T) {
	t.Run("when the status code is not in the whitelist, we should return false", func(t *testing.T) {
		whitelist := []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
		}

		actual := tryAgain(http.StatusInternalServerError, whitelist)

		assert.False(t, actual)
	})

	t.Run("when the status code is in the whitelist, we should return true", func(t *testing.T) {
		whitelist := []int{
			http.StatusTooManyRequests,
			http.StatusBadGateway,
		}

		actual := tryAgain(http.StatusTooManyRequests, whitelist)

		assert.True(t, actual)
	})
}
