package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__WithConstantWait(t *testing.T) {
	// Keep waits at zero so the suite stays fast; we are exercising
	// control flow and attempt counting, not real timing.
	baseOptions := func() Options {
		return Options{
			Task:         "test-task",
			MaxAttempts:  3,
			Wait:         0,
			InitialDelay: 0,
			Verbose:      false,
		}
	}

	t.Run("succeeds on first attempt", func(t *testing.T) {
		calls := 0
		err := WithConstantWait(func() error {
			calls++
			return nil
		}, baseOptions())

		require.NoError(t, err)
		assert.Equal(t, 1, calls, "task should run exactly once when it succeeds immediately")
	})

	t.Run("succeeds after transient failures", func(t *testing.T) {
		calls := 0
		err := WithConstantWait(func() error {
			calls++
			if calls < 3 {
				return errors.New("transient failure")
			}
			return nil
		}, baseOptions())

		require.NoError(t, err)
		assert.Equal(t, 3, calls, "task should stop retrying as soon as it succeeds")
	})

	t.Run("returns wrapped error after exhausting attempts", func(t *testing.T) {
		calls := 0
		err := WithConstantWait(func() error {
			calls++
			return errors.New("permanent failure")
		}, baseOptions())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "test-task")
		assert.Contains(t, err.Error(), "giving up")
		assert.Contains(t, err.Error(), "permanent failure")

		// NOTE: This documents actual current behavior. The loop starts at
		// attempt=1, runs the task, and only gives up once attempt exceeds
		// MaxAttempts. With MaxAttempts=3 the task is therefore invoked 4
		// times before returning an error (attempts 1-4). This test captures
		// that behavior rather than changing it.
		assert.Equal(t, 4, calls, "task runs MaxAttempts+1 times before giving up")
	})

	t.Run("respects InitialDelay and Wait durations", func(t *testing.T) {
		options := baseOptions()
		options.MaxAttempts = 1
		options.InitialDelay = 5 * time.Millisecond
		options.Wait = 5 * time.Millisecond

		start := time.Now()
		calls := 0
		err := WithConstantWait(func() error {
			calls++
			return errors.New("always fails")
		}, options)
		elapsed := time.Since(start)

		require.Error(t, err)
		// With MaxAttempts=1 the task runs twice (attempts 1 and 2). Each
		// iteration sleeps InitialDelay before the task; the first failing
		// attempt also sleeps Wait before retrying. Total delay is therefore
		// at least InitialDelay*2 + Wait = 15ms.
		assert.Equal(t, 2, calls)
		assert.GreaterOrEqual(t, elapsed, 15*time.Millisecond)
	})

	t.Run("verbose logging does not change outcome", func(t *testing.T) {
		options := baseOptions()
		options.Verbose = true

		calls := 0
		err := WithConstantWait(func() error {
			calls++
			if calls < 2 {
				return errors.New("one failure")
			}
			return nil
		}, options)

		require.NoError(t, err)
		assert.Equal(t, 2, calls)
	})
}
