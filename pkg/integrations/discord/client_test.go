package discord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__HTTPClient__IsBounded(t *testing.T) {
	t.Run("shared clients have a request timeout", func(t *testing.T) {
		assert.Equal(t, requestTimeout, httpClient.Timeout)
		assert.Positive(t, httpClient.Timeout, "requests must not be able to hang forever")

		assert.Equal(t, uploadTimeout, uploadHTTPClient.Timeout)
		assert.Positive(t, uploadHTTPClient.Timeout, "uploads must not be able to hang forever")
	})

	t.Run("uploads get more time than plain API calls", func(t *testing.T) {
		assert.Greater(t, uploadHTTPClient.Timeout, httpClient.Timeout)
	})

	t.Run("response size limit is set", func(t *testing.T) {
		assert.Positive(t, maxResponseSize)
	})
}
