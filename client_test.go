package again

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("when the client options are empty, we should return a client with default values", func(t *testing.T) {
		client := NewClient(3)

		trn := client.Transport.(*retryTransport)

		assert.NotNil(t, client.Transport)
		assert.Equal(t, DefaultWhitelist, trn.whitelist)
		assert.Equal(t, 3, trn.maxRetries)
	})
}
