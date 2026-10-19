package grpcerrors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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

func TestInternal(t *testing.T) {
	t.Run("preserves message and code", func(t *testing.T) {
		code, msg, ok := HandlerStatus(Internal(errors.New("db down"), "failed to get user roles"))
		require.True(t, ok)
		assert.Equal(t, codes.Internal, code)
		assert.Equal(t, "failed to get user roles", msg)
	})

	t.Run("unwraps for errors.Is", func(t *testing.T) {
		assert.ErrorIs(t, Internal(context.Canceled, "failed to get user roles"), context.Canceled)
	})

	t.Run("nil err", func(t *testing.T) {
		assert.Nil(t, Internal(nil, "ignored"))
	})
}

func TestNotFound(t *testing.T) {
	code, msg, ok := HandlerStatus(NotFound(gorm.ErrRecordNotFound, "user not found"))
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, code)
	assert.Equal(t, "user not found", msg)
}

func TestStatusFromContextErrorWithInternal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok, statusErr := StatusFromContextError(ctx, Internal(context.Canceled, "failed to get user roles"))
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, status.Code(statusErr))
}

func TestHandlerMessage(t *testing.T) {
	_, ok := HandlerMessage(gorm.ErrRecordNotFound)
	assert.False(t, ok)
}

func TestCode(t *testing.T) {
	assert.Equal(t, codes.NotFound, Code(NotFound(gorm.ErrRecordNotFound, "user not found")))
	assert.Equal(t, codes.Internal, Code(Internal(errors.New("db down"), "failed")))
	assert.Equal(t, codes.PermissionDenied, Code(status.Error(codes.PermissionDenied, "nope")))
}

func TestSanitizeError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.NoError(t, SanitizeError(context.Background(), nil))
	})

	t.Run("grpc status passes through", func(t *testing.T) {
		original := status.Error(codes.PermissionDenied, "nope")
		assert.Equal(t, original, SanitizeError(context.Background(), original))
	})

	t.Run("canceled request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := SanitizeError(ctx, context.Canceled)
		require.Error(t, err)
		assert.Equal(t, codes.Canceled, status.Code(err))
		assert.Equal(t, RequestCanceledMessage, status.Convert(err).Message())
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		err := SanitizeError(context.Background(), context.DeadlineExceeded)
		require.Error(t, err)
		assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
		assert.Equal(t, RequestDeadlineExceededMessage, status.Convert(err).Message())
	})

	t.Run("handler message", func(t *testing.T) {
		err := SanitizeError(context.Background(), Internal(errors.New("db down"), "failed to get user roles"))
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "failed to get user roles", status.Convert(err).Message())
	})

	t.Run("record not found", func(t *testing.T) {
		err := SanitizeError(context.Background(), gorm.ErrRecordNotFound)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, "resource not found", status.Convert(err).Message())
	})

	t.Run("not found text", func(t *testing.T) {
		err := SanitizeError(context.Background(), errors.New("Widget Not Found"))
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, "resource not found", status.Convert(err).Message())
	})

	t.Run("postgres error", func(t *testing.T) {
		err := SanitizeError(context.Background(), &pgconn.PgError{Code: "23505", Message: "duplicate key"})
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "internal error", status.Convert(err).Message())
	})

	t.Run("other errors", func(t *testing.T) {
		err := SanitizeError(context.Background(), errors.New("db down"))
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, "internal error", status.Convert(err).Message())
	})
}

func TestStatusMessage(t *testing.T) {
	assert.Equal(t, "user not found", StatusMessage(NotFound(gorm.ErrRecordNotFound, "user not found")))
	assert.Equal(t, "organization canvas limit exceeded", StatusMessage(status.Error(codes.ResourceExhausted, "organization canvas limit exceeded")))
	assert.Empty(t, StatusMessage(nil))
}
