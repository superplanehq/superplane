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

	t.Run("non-409 with 'already' in message is not a conflict", func(t *testing.T) {
		err := &CloudflareAPIError{
			StatusCode: http.StatusBadRequest,
			Errors:     []CloudflareError{{Code: 10009, Message: "A worker with this name already exists"}},
			Body:       []byte(`{"success":false}`),
		}
		assert.False(t, isWorkerProvisionConflict(err))
	})

	t.Run("token already revoked is not a conflict", func(t *testing.T) {
		err := &CloudflareAPIError{
			StatusCode: http.StatusUnauthorized,
			Errors:     []CloudflareError{{Code: 9109, Message: "Token already revoked"}},
			Body:       []byte(`{"success":false}`),
		}
		assert.False(t, isWorkerProvisionConflict(err))
	})

	t.Run("quota already exhausted is not a conflict", func(t *testing.T) {
		err := &CloudflareAPIError{
			StatusCode: http.StatusTooManyRequests,
			Errors:     []CloudflareError{{Code: 10000, Message: "Quota already exhausted"}},
			Body:       []byte(`{"success":false}`),
		}
		assert.False(t, isWorkerProvisionConflict(err))
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
