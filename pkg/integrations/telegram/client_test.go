package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__HTTPClient__IsBounded(t *testing.T) {
	t.Run("shared client has a request timeout", func(t *testing.T) {
		assert.Equal(t, requestTimeout, httpClient.Timeout)
		assert.Positive(t, httpClient.Timeout, "requests must not be able to hang forever")
	})

	t.Run("response size limit is set", func(t *testing.T) {
		assert.Positive(t, maxResponseSize)
	})
}
