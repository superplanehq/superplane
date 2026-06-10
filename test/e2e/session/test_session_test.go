package session

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestIsRetryableResetDatabaseError(t *testing.T) {
	t.Run("deadlock detected postgres error", func(t *testing.T) {
		err := fmt.Errorf("reset failed: %w", &pgconn.PgError{Code: postgresDeadlockDetectedCode})

		require.True(t, isRetryableResetDatabaseError(err))
	})

	t.Run("lock timeout postgres error", func(t *testing.T) {
		err := &pgconn.PgError{Code: postgresLockNotAvailableCode}

		require.True(t, isRetryableResetDatabaseError(err))
	})

	t.Run("deadlock detected log message", func(t *testing.T) {
		err := errors.New("ERROR: deadlock detected (SQLSTATE 40P01)")

		require.True(t, isRetryableResetDatabaseError(err))
	})

	t.Run("non retryable error", func(t *testing.T) {
		err := errors.New("syntax error")

		require.False(t, isRetryableResetDatabaseError(err))
	})
}
