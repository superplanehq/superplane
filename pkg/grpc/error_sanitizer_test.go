package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestSanitizeError(t *testing.T) {
	t.Run("client canceled request maps to canceled status", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := SanitizeError(ctx, context.Canceled)
		require.Error(t, err)
		assert.Equal(t, codes.Canceled, status.Code(err))
	})

	t.Run("existing grpc status passes through", func(t *testing.T) {
		original := status.Error(codes.PermissionDenied, "nope")
		assert.Equal(t, original, SanitizeError(context.Background(), original))
	})

	t.Run("record not found maps to not found", func(t *testing.T) {
		err := SanitizeError(context.Background(), gorm.ErrRecordNotFound)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("other errors map to internal", func(t *testing.T) {
		err := SanitizeError(context.Background(), errors.New("db down"))
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}
