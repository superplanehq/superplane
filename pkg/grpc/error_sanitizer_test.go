package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
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
		assert.Equal(t, sanitizedInternalMessage, status.Convert(err).Message())
	})

	t.Run("internal error preserves handler message", func(t *testing.T) {
		err := SanitizeError(context.Background(), grpcerrors.Internal(errors.New("db down"), "failed to get user roles"))
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "failed to get user roles", status.Convert(err).Message())
	})

	t.Run("not found error preserves handler message", func(t *testing.T) {
		err := SanitizeError(context.Background(), grpcerrors.NotFound(gorm.ErrRecordNotFound, "user not found"))
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, "user not found", status.Convert(err).Message())
	})

	t.Run("internal cancel maps to canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := SanitizeError(ctx, grpcerrors.Internal(context.Canceled, "failed to get user roles"))
		assert.Equal(t, codes.Canceled, status.Code(err))
	})

	t.Run("postgres query_canceled maps to canceled when request ctx is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		pgErr := &pgconn.PgError{Code: pgQueryCanceledCode, Message: "canceling statement due to user request"}

		err := SanitizeError(ctx, grpcerrors.Internal(pgErr, "failed to list integrations"))
		assert.Equal(t, codes.Canceled, status.Code(err))
		assert.Equal(t, grpcerrors.RequestCanceledMessage, status.Convert(err).Message())
	})

	t.Run("postgres query_canceled maps to deadline exceeded when request ctx is deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		<-ctx.Done()

		pgErr := &pgconn.PgError{Code: pgQueryCanceledCode, Message: "canceling statement due to statement timeout"}

		err := SanitizeError(ctx, grpcerrors.Internal(pgErr, "failed to list integrations"))
		assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
		assert.Equal(t, grpcerrors.RequestDeadlineExceededMessage, status.Convert(err).Message())
	})

	t.Run("postgres query_canceled keeps handler internal when request ctx is alive", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: pgQueryCanceledCode, Message: "canceling statement due to statement timeout"}

		err := SanitizeError(context.Background(), grpcerrors.Internal(pgErr, "failed to list integrations"))
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "failed to list integrations", status.Convert(err).Message())
	})

	t.Run("postgres query_canceled wrapped via fmt.Errorf still maps to canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		pgErr := &pgconn.PgError{Code: pgQueryCanceledCode, Message: "canceling statement due to user request"}
		wrapped := fmt.Errorf("query failed: %w", pgErr)

		err := SanitizeError(ctx, wrapped)
		assert.Equal(t, codes.Canceled, status.Code(err))
	})

	t.Run("other pg errors remain internal even when request ctx is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}

		err := SanitizeError(ctx, grpcerrors.Internal(pgErr, "failed to create thing"))
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}
