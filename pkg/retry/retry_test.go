package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__WithConstantWait(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		attempts := 0
		err := WithConstantWait(func() error {
			attempts++
			return nil
		}, Options{Task: "test", MaxAttempts: 3})

		require.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries until it succeeds", func(t *testing.T) {
		attempts := 0
		err := WithConstantWait(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("not yet")
			}
			return nil
		}, Options{Task: "test", MaxAttempts: 5})

		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("gives up after MaxAttempts and wraps the last error", func(t *testing.T) {
		attempts := 0
		err := WithConstantWait(func() error {
			attempts++
			return errors.New("boom")
		}, Options{Task: "flaky", MaxAttempts: 2})

		require.Error(t, err)
		assert.Equal(t, 3, attempts)
		assert.Contains(t, err.Error(), "flaky")
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("waits between attempts", func(t *testing.T) {
		attempts := 0
		start := time.Now()
		err := WithConstantWait(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("retry")
			}
			return nil
		}, Options{Task: "test", MaxAttempts: 3, Wait: 20 * time.Millisecond})

		require.NoError(t, err)
		assert.GreaterOrEqual(t, time.Since(start), 20*time.Millisecond)
	})

	t.Run("verbose logging does not change the outcome", func(t *testing.T) {
		attempts := 0
		err := WithConstantWait(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("retry")
			}
			return nil
		}, Options{Task: "test", MaxAttempts: 3, Verbose: true})

		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})
}
