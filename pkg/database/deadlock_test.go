package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestRetryOnDeadlock(t *testing.T) {
	t.Run("retries a deadlock until the operation succeeds", func(t *testing.T) {
		calls := 0
		waits := 0
		err := RetryOnDeadlock(3, func() error {
			calls++
			if calls < 3 {
				return errors.New("ERROR: deadlock detected (SQLSTATE 40P01)")
			}
			return nil
		}, func(int) {
			waits++
		})

		require.NoError(t, err)
		require.Equal(t, 3, calls)
		require.Equal(t, 2, waits)
	})

	t.Run("retries a wrapped postgres deadlock", func(t *testing.T) {
		calls := 0
		err := RetryOnDeadlock(2, func() error {
			calls++
			if calls == 1 {
				return fmt.Errorf("reset database: %w", &pgconn.PgError{Code: "40P01", Message: "deadlock detected"})
			}
			return nil
		}, nil)

		require.NoError(t, err)
		require.Equal(t, 2, calls)
	})

	t.Run("does not retry other errors", func(t *testing.T) {
		calls := 0
		err := RetryOnDeadlock(3, func() error {
			calls++
			return errors.New("connection refused")
		}, func(int) {
			t.Fatal("waited for a non-deadlock error")
		})

		require.EqualError(t, err, "connection refused")
		require.Equal(t, 1, calls)
	})

	t.Run("returns the last deadlock", func(t *testing.T) {
		calls := 0
		err := RetryOnDeadlock(2, func() error {
			calls++
			return &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
		}, func(int) {})

		require.Error(t, err)
		require.Contains(t, err.Error(), "deadlock detected")
		require.Equal(t, 2, calls)
	})
}
