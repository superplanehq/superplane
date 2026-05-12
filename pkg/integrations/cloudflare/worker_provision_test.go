package cloudflare

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__isWorkerProvisionConflict(t *testing.T) {
	t.Run("409 is conflict", func(t *testing.T) {
		err := &CloudflareAPIError{StatusCode: http.StatusConflict, Body: []byte(`{}`)}
		assert.True(t, isWorkerProvisionConflict(err))
	})

	t.Run("already in message", func(t *testing.T) {
		err := &CloudflareAPIError{
			StatusCode: http.StatusBadRequest,
			Errors:     []CloudflareError{{Code: 10009, Message: "A worker with this name already exists"}},
			Body:       []byte(`{"success":false}`),
		}
		assert.True(t, isWorkerProvisionConflict(err))
	})

	t.Run("unrelated error is not conflict", func(t *testing.T) {
		err := &CloudflareAPIError{
			StatusCode: http.StatusBadRequest,
			Errors:     []CloudflareError{{Code: 10001, Message: "Invalid request"}},
			Body:       []byte(`{}`),
		}
		assert.False(t, isWorkerProvisionConflict(err))
	})

	t.Run("non API error", func(t *testing.T) {
		assert.False(t, isWorkerProvisionConflict(errors.New("network")))
	})
}
