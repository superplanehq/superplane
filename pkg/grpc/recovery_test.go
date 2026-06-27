package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSentryRecoveryHandler(t *testing.T) {
	t.Run("returns an Internal gRPC status without panicking", func(t *testing.T) {
		err := sentryRecoveryHandler("boom")

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "internal server error")
	})

	t.Run("handles a nil interface panic value", func(t *testing.T) {
		err := sentryRecoveryHandler(nil)

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})

	t.Run("handles a runtime error panic value", func(t *testing.T) {
		var typeAssertionPanic any
		func() {
			defer func() {
				typeAssertionPanic = recover()
			}()
			var something any
			_ = something.(string)
		}()
		require.NotNil(t, typeAssertionPanic)

		err := sentryRecoveryHandler(typeAssertionPanic)

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}
