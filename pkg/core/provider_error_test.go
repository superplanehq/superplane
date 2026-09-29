package core

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__ProviderAPIError__Classification(t *testing.T) {
	t.Run("auth errors", func(t *testing.T) {
		for _, code := range []int{401, 403} {
			err := NewProviderAPIError(code, "credentials invalid", fmt.Errorf("boom"))
			assert.True(t, err.IsAuth(), "status %d should classify as auth", code)
			assert.False(t, err.IsRateLimited())
			assert.False(t, err.IsUnavailable())
			assert.False(t, err.IsTransport())
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		err := NewProviderAPIError(429, "too many requests", fmt.Errorf("boom"))
		assert.True(t, err.IsRateLimited())
		assert.False(t, err.IsAuth())
		assert.False(t, err.IsUnavailable())
	})

	t.Run("unavailable", func(t *testing.T) {
		for _, code := range []int{500, 502, 503} {
			err := NewProviderAPIError(code, "server error", fmt.Errorf("boom"))
			assert.True(t, err.IsUnavailable(), "status %d should classify as unavailable", code)
		}
	})

	t.Run("transport failure", func(t *testing.T) {
		err := NewProviderTransportError("request failed: connection reset", fmt.Errorf("boom"))
		assert.True(t, err.IsTransport())
		assert.False(t, err.IsAuth())
		assert.Equal(t, 0, err.StatusCode)
	})

	t.Run("Error() text is preserved", func(t *testing.T) {
		err := NewProviderAPIError(401, "Claude credentials are invalid or expired: bad key", fmt.Errorf("boom"))
		assert.Equal(t, "Claude credentials are invalid or expired: bad key", err.Error())
	})
}

func Test__IsProviderAuthOrNetworkError(t *testing.T) {
	t.Run("typed auth error", func(t *testing.T) {
		err := NewProviderAPIError(401, "invalid key", fmt.Errorf("boom"))
		assert.True(t, IsProviderAuthOrNetworkError(err))
	})

	t.Run("typed transport error", func(t *testing.T) {
		err := NewProviderTransportError("request failed", fmt.Errorf("boom"))
		assert.True(t, IsProviderAuthOrNetworkError(err))
	})

	t.Run("typed unavailable error", func(t *testing.T) {
		err := NewProviderAPIError(503, "unavailable", fmt.Errorf("boom"))
		assert.True(t, IsProviderAuthOrNetworkError(err))
	})

	t.Run("typed rate limited error is not auth/network", func(t *testing.T) {
		err := NewProviderAPIError(429, "rate limited", fmt.Errorf("boom"))
		assert.False(t, IsProviderAuthOrNetworkError(err))
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		assert.True(t, IsProviderAuthOrNetworkError(context.DeadlineExceeded))
	})

	t.Run("net timeout error", func(t *testing.T) {
		assert.True(t, IsProviderAuthOrNetworkError(&net.DNSError{IsTimeout: true}))
	})

	t.Run("untyped error", func(t *testing.T) {
		assert.False(t, IsProviderAuthOrNetworkError(fmt.Errorf("bug in our code")))
	})
}

func Test__IsProviderRateLimited(t *testing.T) {
	t.Run("typed rate limited error", func(t *testing.T) {
		err := NewProviderAPIError(429, "rate limited", fmt.Errorf("boom"))
		assert.True(t, IsProviderRateLimited(err))
	})

	t.Run("typed auth error is not rate limited", func(t *testing.T) {
		err := NewProviderAPIError(401, "invalid key", fmt.Errorf("boom"))
		assert.False(t, IsProviderRateLimited(err))
	})

	t.Run("untyped error", func(t *testing.T) {
		assert.False(t, IsProviderRateLimited(fmt.Errorf("boom")))
	})
}
