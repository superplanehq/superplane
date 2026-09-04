package retry

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__WithConstantWait_SuccessOnFirstAttempt(t *testing.T) {
	var calls atomic.Int32
	err := WithConstantWait(func() error {
		calls.Add(1)
		return nil
	}, Options{
		Task:        "test",
		MaxAttempts: 3,
		Wait:        time.Millisecond,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func Test__WithConstantWait_SuccessAfterRetry(t *testing.T) {
	var calls atomic.Int32
	err := WithConstantWait(func() error {
		calls.Add(1)
		if calls.Load() < 3 {
			return errors.New("not yet")
		}
		return nil
	}, Options{
		Task:        "test",
		MaxAttempts: 5,
		Wait:        time.Millisecond,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func Test__WithConstantWait_ExhaustsRetries(t *testing.T) {
	const maxAttempts = 3
	var calls atomic.Int32
	err := WithConstantWait(func() error {
		calls.Add(1)
		return errors.New("always fails")
	}, Options{
		Task:        "test",
		MaxAttempts: maxAttempts,
		Wait:        time.Millisecond,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test")
	assert.Contains(t, err.Error(), "failed after")
	assert.Contains(t, err.Error(), "always fails")
	assert.Equal(t, int32(maxAttempts+1), calls.Load())
}

func Test__WithConstantWait_ZeroAttemptsDefaultsToSingleTry(t *testing.T) {
	var calls atomic.Int32
	err := WithConstantWait(func() error {
		calls.Add(1)
		return errors.New("fail")
	}, Options{
		Task: "test",
		Wait: time.Millisecond,
	})
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func Test__WithConstantWait_InitialDelay(t *testing.T) {
	start := time.Now()
	_ = WithConstantWait(func() error { return nil }, Options{
		Task:         "test",
		MaxAttempts:  1,
		InitialDelay: 5 * time.Millisecond,
	})
	assert.GreaterOrEqual(t, time.Since(start), 5*time.Millisecond)
}

func Test__WithConstantWait_VerboseDoesNotPanic(t *testing.T) {
	err := WithConstantWait(func() error {
		return errors.New("fail")
	}, Options{
		Task:        "test",
		MaxAttempts: 1,
		Wait:        time.Millisecond,
		Verbose:     true,
	})
	require.Error(t, err)
}

func Test__WithConstantWait_ReturnsNilWhenNoError(t *testing.T) {
	err := WithConstantWait(func() error { return nil }, Options{
		Task:        "noop",
		MaxAttempts: 0,
	})
	assert.NoError(t, err)
}

func Test__WithConstantWait_ZeroInitialDelayNoSleep(t *testing.T) {
	start := time.Now()
	_ = WithConstantWait(func() error { return nil }, Options{
		Task: "fast",
	})
	assert.WithinDuration(t, start, time.Now(), time.Millisecond)
}
