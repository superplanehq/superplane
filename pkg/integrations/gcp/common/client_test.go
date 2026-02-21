package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseGCPError(t *testing.T) {
	t.Run("uses API error message when present", func(t *testing.T) {
		body := []byte(`{"error":{"code":404,"message":"Not found","status":"NOT_FOUND"}}`)
		err := ParseGCPError(404, body)
		require.Error(t, err)
		var apiErr *GCPAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 404, apiErr.StatusCode)
		assert.Equal(t, "Not found", apiErr.Message)
	})

	t.Run("falls back to raw body when not JSON", func(t *testing.T) {
		body := []byte("plain text error")
		err := ParseGCPError(500, body)
		require.Error(t, err)
		var apiErr *GCPAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 500, apiErr.StatusCode)
		assert.Equal(t, "plain text error", apiErr.Message)
	})

	t.Run("empty error message uses body", func(t *testing.T) {
		body := []byte(`{"error":{}}`)
		err := ParseGCPError(403, body)
		require.Error(t, err)
		var apiErr *GCPAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 403, apiErr.StatusCode)
		assert.Equal(t, `{"error":{}}`, apiErr.Message)
	})
}

func Test_GCPAPIError_Error(t *testing.T) {
	err := &GCPAPIError{StatusCode: 404, Message: "Not found"}
	assert.Equal(t, "GCP request failed (404): Not found", err.Error())
}
