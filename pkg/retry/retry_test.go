package retry

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// MaxAttempts is checked with `attempt > options.MaxAttempts` *after* the task
// has already run, so the task executes MaxAttempts+1 times before the helper
// gives up. Pinning that down so the effective retry budget cannot drift
// unnoticed if the loop is ever restructured again.
func Test__WithConstantWait__RunsMaxAttemptsPlusOneBeforeGivingUp(t *testing.T) {
	for _, maxAttempts := range []int{0, 1, 3} {
		calls := 0
		err := WithConstantWait(func() error {
			calls++
			return errors.New("boom")
		}, Options{Task: "test", MaxAttempts: maxAttempts})

		assert.Error(t, err, "MaxAttempts=%d", maxAttempts)
		assert.Equal(t, maxAttempts+1, calls, "MaxAttempts=%d", maxAttempts)
	}
}

func Test__WithConstantWait__ErrorReportsTaskAttemptsAndCause(t *testing.T) {
	cause := errors.New("connection refused")

	err := WithConstantWait(func() error {
		return cause
	}, Options{Task: "fetch-webhook", MaxAttempts: 2})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "[fetch-webhook]")
	// Three runs for MaxAttempts=2, per the off-by-one documented above.
	assert.Contains(t, err.Error(), "[3]")
	assert.Contains(t, err.Error(), cause.Error())
}

// The caller only ever sees the failure from the final run, so a task whose
// error changes between attempts must not report a stale one.
func Test__WithConstantWait__ReportsLastErrorNotFirst(t *testing.T) {
	calls := 0

	err := WithConstantWait(func() error {
		calls++
		return fmt.Errorf("failure %d", calls)
	}, Options{Task: "test", MaxAttempts: 2})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failure 3")
	assert.NotContains(t, err.Error(), "failure 1")
}

// InitialDelay is documented as a one-time delay before the first attempt.
// This guards against it being applied before every attempt (which would
// stack with Wait on each retry).
//
// The assertions compare the gaps *between* task runs rather than the total
// elapsed time: a loaded machine can stretch the first sleep well past its
// nominal duration, which would make a total-elapsed upper bound flaky while
// telling us nothing extra about where the delay was spent.
func Test__WithConstantWait__InitialDelayAppliedOnce(t *testing.T) {
	initialDelay := 50 * time.Millisecond

	var runs []time.Time
	start := time.Now()
	err := WithConstantWait(func() error {
		runs = append(runs, time.Now())
		if len(runs) < 3 {
			return errors.New("boom")
		}
		return nil
	}, Options{Task: "test", MaxAttempts: 5, InitialDelay: initialDelay})

	require.NoError(t, err)
	require.Len(t, runs, 3)

	// The delay is applied before the first attempt...
	assert.GreaterOrEqual(t, runs[0].Sub(start), initialDelay)

	// ...and never again. Wait is zero here, so each retry should follow the
	// previous run almost immediately.
	for i := 1; i < len(runs); i++ {
		gap := runs[i].Sub(runs[i-1])
		assert.Less(t, gap, initialDelay,
			"attempt %d started %s after the previous one - InitialDelay must not be re-applied", i+1, gap)
	}
}

// The only production caller leaves InitialDelay at its zero value, so the
// happy path must not pay any startup cost.
func Test__WithConstantWait__NoDelayWhenInitialDelayUnset(t *testing.T) {
	start := time.Now()
	err := WithConstantWait(func() error {
		return nil
	}, Options{Task: "test", MaxAttempts: 3})

	assert.NoError(t, err)
	assert.Less(t, time.Since(start), 50*time.Millisecond)
}

func Test__WithConstantWait__WaitAppliedBetweenRetries(t *testing.T) {
	wait := 30 * time.Millisecond

	var runs []time.Time
	err := WithConstantWait(func() error {
		runs = append(runs, time.Now())
		if len(runs) < 3 {
			return errors.New("boom")
		}
		return nil
	}, Options{Task: "test", MaxAttempts: 5, Wait: wait})

	require.NoError(t, err)
	require.Len(t, runs, 3)

	for i := 1; i < len(runs); i++ {
		assert.GreaterOrEqual(t, runs[i].Sub(runs[i-1]), wait,
			"retry %d should be spaced by at least Wait", i+1)
	}
}

func Test__WithConstantWait__VerboseDoesNotChangeOutcome(t *testing.T) {
	calls := 0
	err := WithConstantWait(func() error {
		calls++
		if calls < 2 {
			return errors.New("boom")
		}
		return nil
	}, Options{Task: "test", MaxAttempts: 3, Verbose: true})

	assert.NoError(t, err)
	assert.Equal(t, 2, calls)
}
