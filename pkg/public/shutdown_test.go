package public

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ServerShutdown(t *testing.T) {
	t.Run("is a no-op before Serve", func(t *testing.T) {
		server := &Server{}
		require.NoError(t, server.Shutdown(context.Background()))
	})

	t.Run("stops Serve from listening after an early Shutdown", func(t *testing.T) {
		server := &Server{}
		require.NoError(t, server.Shutdown(context.Background()))

		//
		// Shutdown ran before Serve, so it had no listener to close. Serve must
		// refuse to start, otherwise it blocks until the process is killed and
		// the drain waits for a goroutine that can never return.
		//
		done := make(chan error, 1)
		go func() { done <- server.Serve("127.0.0.1", 0) }()

		select {
		case err := <-done:
			assert.ErrorIs(t, err, http.ErrServerClosed)
		case <-time.After(5 * time.Second):
			require.Fail(t, "Serve started listening after Shutdown")
		}
	})

	t.Run("Close is a no-op before Serve", func(t *testing.T) {
		server := &Server{}
		assert.NotPanics(t, server.Close)
	})
}
