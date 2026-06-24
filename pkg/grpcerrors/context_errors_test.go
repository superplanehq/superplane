package grpcerrors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusFromContextError(t *testing.T) {
	t.Run("client canceled request context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ok, statusErr := StatusFromContextError(ctx, context.Canceled)
		require.True(t, ok)
		assert.Equal(t, codes.Canceled, status.Code(statusErr))
		assert.Equal(t, RequestCanceledMessage, status.Convert(statusErr).Message())
	})

	t.Run("handler deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		<-ctx.Done()

		ok, statusErr := StatusFromContextError(context.Background(), ctx.Err())
		require.True(t, ok)
		assert.Equal(t, codes.DeadlineExceeded, status.Code(statusErr))
	})

	t.Run("non context error", func(t *testing.T) {
		ok, statusErr := StatusFromContextError(context.Background(), errors.New("boom"))
		assert.False(t, ok)
		assert.Nil(t, statusErr)
	})
}
