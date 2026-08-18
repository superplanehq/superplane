package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test__WithConstantWait__SucceedsOnFirstAttempt(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		return nil
	}, Options{Task: "test", MaxAttempts: 3})

	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func Test__WithConstantWait__RetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		if calls < 3 {
			return errors.New("boom")
		}
		return nil
	}, Options{Task: "test", MaxAttempts: 5})

	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func Test__WithConstantWait__ReturnsErrorAfterGivingUp(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		return errors.New("boom")
	}, Options{Task: "test", MaxAttempts: 2})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "giving up")
}

// InitialDelay is documented as a one-time delay before the first attempt.
// This guards against it being applied before every attempt (which would
// stack with Wait on each retry).
func Test__WithConstantWait__InitialDelayAppliedOnce(t *testing.T) {
	initialDelay := 50 * time.Millisecond
	calls := 0

	start := time.Now()
	err := WithConstantWait(func() error {
		calls++
		if calls < 3 {
			return errors.New("boom")
		}
		return nil
	}, Options{Task: "test", MaxAttempts: 5, InitialDelay: initialDelay})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
	// The initial delay must have been waited at least once...
	assert.GreaterOrEqual(t, elapsed, initialDelay)
	// ...but not once per attempt (3 attempts would be >= 150ms).
	assert.Less(t, elapsed, 3*initialDelay)
}
