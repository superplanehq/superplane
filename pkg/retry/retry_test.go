package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithConstantWait_SucceedsOnFirstAttempt(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		return nil
	}, Options{Task: "t", MaxAttempts: 3, Wait: time.Millisecond})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestWithConstantWait_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	}, Options{Task: "t", MaxAttempts: 5, Wait: time.Millisecond})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestWithConstantWait_GivesUpAfterMaxAttempts(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		return errors.New("always fails")
	}, Options{Task: "my-task", MaxAttempts: 2, Wait: time.Millisecond})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "my-task")
	assert.Contains(t, err.Error(), "always fails")
	// MaxAttempts=2 allows the initial attempt plus 2 retries before giving up.
	assert.Equal(t, 3, calls)
}

// InitialDelay must only be applied once, before the first attempt — not
// re-applied before every subsequent retry on top of Wait. Regression test
// for a bug where the delay compounded with Wait on every retry.
func TestWithConstantWait_InitialDelayAppliedOnlyOnce(t *testing.T) {
	calls := 0
	start := time.Now()

	err := WithConstantWait(func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	}, Options{
		Task:         "t",
		MaxAttempts:  5,
		Wait:         5 * time.Millisecond,
		InitialDelay: 50 * time.Millisecond,
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 3, calls)

	// Expected: one InitialDelay (50ms) + two Wait intervals (2*5ms) = ~60ms.
	// Before the fix, InitialDelay was re-slept before every attempt too,
	// which would push this past ~150ms.
	assert.Less(t, elapsed, 100*time.Millisecond)
	assert.GreaterOrEqual(t, elapsed, 60*time.Millisecond)
}
