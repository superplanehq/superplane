package retry

import (
	"errors"
	"slices"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestWithConstantWait(t *testing.T) {
	cases := []struct {
		name        string
		maxAttempts int
		failures    int
		wantErr     string
		wantCalls   int
	}{
		{
			name:        "succeeds first try",
			maxAttempts: 3,
			failures:    0,
			wantCalls:   1,
		},
		{
			name:        "succeeds after retries",
			maxAttempts: 3,
			failures:    2,
			wantCalls:   3,
		},
		{
			name:        "gives up after max attempts",
			maxAttempts: 2,
			failures:    99,
			wantErr:     "[mytask] failed after [3] attempts - giving up",
			wantCalls:   3,
		},
		{
			name:        "zero max attempts still runs once",
			maxAttempts: 0,
			failures:    99,
			wantErr:     "[mytask] failed after [1] attempts - giving up",
			wantCalls:   1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := 0
			err := WithConstantWait(func() error {
				calls++
				if calls <= c.failures {
					return errors.New("boom")
				}
				return nil
			}, Options{
				Task:        "mytask",
				MaxAttempts: c.maxAttempts,
			})

			require.Equal(t, c.wantCalls, calls)
			if c.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), c.wantErr)
			require.Contains(t, err.Error(), "boom")
		})
	}
}

func TestWithConstantWaitVerboseLogsAttempt(t *testing.T) {
	hook := logtest.NewLocal(log.StandardLogger())
	t.Cleanup(hook.Reset)

	err := WithConstantWait(
		func() error { return errors.New("boom") },
		Options{Task: "mytask", MaxAttempts: 1, Verbose: true},
	)
	require.Error(t, err)

	require.True(t, slices.ContainsFunc(
		hook.AllEntries(),
		func(entry *log.Entry) bool {
			return strings.Contains(entry.Message, "attempt [1] failed with [boom]")
		},
	), "expected verbose attempt log")
}
