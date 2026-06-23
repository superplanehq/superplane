package me

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test_translateMeError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.NoError(t, translateMeError(nil, codes.Internal, "should not be used"))
	})

	t.Run("existing status error is preserved", func(t *testing.T) {
		original := status.Error(codes.PermissionDenied, "nope")

		got := translateMeError(original, codes.Internal, "fallback")

		assert.Equal(t, codes.PermissionDenied, status.Code(got))
		assert.Equal(t, "nope", status.Convert(got).Message())
	})

	t.Run("context.Canceled maps to codes.Canceled", func(t *testing.T) {
		got := translateMeError(context.Canceled, codes.Internal, "fallback")

		assert.Equal(t, codes.Canceled, status.Code(got))
	})

	t.Run("wrapped context.Canceled maps to codes.Canceled", func(t *testing.T) {
		got := translateMeError(fmt.Errorf("wrapped: %w", context.Canceled), codes.Internal, "fallback")

		assert.Equal(t, codes.Canceled, status.Code(got))
	})

	t.Run("context.DeadlineExceeded maps to codes.DeadlineExceeded", func(t *testing.T) {
		got := translateMeError(context.DeadlineExceeded, codes.Internal, "fallback")

		assert.Equal(t, codes.DeadlineExceeded, status.Code(got))
	})

	t.Run("wrapped context.DeadlineExceeded maps to codes.DeadlineExceeded", func(t *testing.T) {
		got := translateMeError(fmt.Errorf("wrapped: %w", context.DeadlineExceeded), codes.Internal, "fallback")

		assert.Equal(t, codes.DeadlineExceeded, status.Code(got))
	})

	t.Run("gorm.ErrRecordNotFound maps to codes.NotFound", func(t *testing.T) {
		got := translateMeError(gorm.ErrRecordNotFound, codes.Internal, "fallback")

		assert.Equal(t, codes.NotFound, status.Code(got))
		assert.Equal(t, "user not found", status.Convert(got).Message())
	})

	t.Run("wrapped gorm.ErrRecordNotFound maps to codes.NotFound", func(t *testing.T) {
		got := translateMeError(fmt.Errorf("wrapped: %w", gorm.ErrRecordNotFound), codes.Internal, "fallback")

		assert.Equal(t, codes.NotFound, status.Code(got))
	})

	t.Run("unknown error falls back to the supplied code and message", func(t *testing.T) {
		got := translateMeError(errors.New("boom"), codes.Internal, "failed to load user")

		assert.Equal(t, codes.Internal, status.Code(got))
		assert.Equal(t, "failed to load user", status.Convert(got).Message())
	})

	t.Run("unknown error uses the supplied non-internal fallback when provided", func(t *testing.T) {
		got := translateMeError(errors.New("boom"), codes.Unavailable, "auth service unavailable")

		assert.Equal(t, codes.Unavailable, status.Code(got))
		assert.Equal(t, "auth service unavailable", status.Convert(got).Message())
	})
}
