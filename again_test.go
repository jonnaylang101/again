package again

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNewClient(t *testing.T) {
	t.Run("when the client options are empty, we should return a client with default values", func(t *testing.T) {
		client := NewClient(3)

		trn := client.Transport.(*retryTransport)

		assert.NotNil(t, client.Transport)
		assert.Equal(t, DefaultWhitelist, trn.whitelist)
		assert.Equal(t, 3, trn.maxRetries)
	})
}
