package checks

import (
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__NormalizePullRequestChecks(t *testing.T) {
	t.Parallel()

	checkRuns := &github.ListCheckRunsResults{
		CheckRuns: []*github.CheckRun{
			{
				Name:       github.Ptr("DCO"),
				Status:     github.Ptr("completed"),
				Conclusion: github.Ptr("success"),
				DetailsURL: github.Ptr("https://example.com/dco"),
				App:        &github.App{Slug: github.Ptr("dco")},
			},
			{
				Name:   github.Ptr("lint"),
				Status: github.Ptr("in_progress"),
				App:    &github.App{Slug: github.Ptr("github-actions")},
			},
			{
				Name:       github.Ptr("DCO"),
				Status:     github.Ptr("completed"),
				Conclusion: github.Ptr("failure"),
				DetailsURL: github.Ptr("https://example.com/dco-later"),
				App:        &github.App{Slug: github.Ptr("dco")},
			},
		},
	}
	combined := &github.CombinedStatus{
		Statuses: []*github.RepoStatus{
			{
				Context:   github.Ptr("ci/semaphore"),
				State:     github.Ptr("pending"),
				TargetURL: github.Ptr("https://example.com/ci"),
			},
			{
				Context:   github.Ptr("ci/semaphore"),
				State:     github.Ptr("success"),
				TargetURL: github.Ptr("https://example.com/ci-later"),
			},
		},
	}

	checks := normalizePullRequestChecks(checkRuns, combined)
	require.Len(t, checks, 3)
	assert.Equal(t, "check-run:dco:DCO", checks[0].Key)
	assert.Equal(t, "failure", checks[0].Conclusion)
	assert.Equal(t, "check-run:github-actions:lint", checks[1].Key)
	assert.Equal(t, checkStatusPending, checks[1].Status)
	assert.Equal(t, "status:ci/semaphore", checks[2].Key)
	assert.Equal(t, "success", checks[2].Conclusion)
}

func Test__EvaluatePullRequestChecks(t *testing.T) {
	t.Parallel()

	passed := PullRequestCheck{Key: "check-run:dco:DCO", Name: "DCO", Status: checkStatusCompleted, Conclusion: "success"}
	failed := PullRequestCheck{Key: "check-run:ci:build", Name: "build", Status: checkStatusCompleted, Conclusion: "failure"}
	pending := PullRequestCheck{Key: "status:ci", Name: "ci", Status: checkStatusPending}

	t.Run("pending while any check is running", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, pending}, nil, false)
		assert.Equal(t, waitChecksOutcomePending, evaluation.Outcome)
		assert.False(t, evaluation.AllTerminal)
	})

	t.Run("failed when a selected check failed", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, failed}, nil, false)
		assert.Equal(t, waitChecksOutcomeFailed, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
		require.Len(t, evaluation.FailedChecks, 1)
		assert.Equal(t, "build", evaluation.FailedChecks[0].Name)
	})

	t.Run("passed when remaining checks were cancelled", func(t *testing.T) {
		t.Parallel()
		cancelled := PullRequestCheck{Key: "check-run:ci:build", Name: "build", Status: checkStatusCompleted, Conclusion: "cancelled"}
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, cancelled}, nil, false)
		assert.Equal(t, waitChecksOutcomePassed, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
		assert.Empty(t, evaluation.FailedChecks)
	})

	t.Run("failed when a check failed among cancelled checks", func(t *testing.T) {
		t.Parallel()
		cancelled := PullRequestCheck{Key: "check-run:ci:lint", Name: "lint", Status: checkStatusCompleted, Conclusion: "cancelled"}
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{failed, cancelled}, nil, false)
		assert.Equal(t, waitChecksOutcomeFailed, evaluation.Outcome)
		require.Len(t, evaluation.FailedChecks, 1)
		assert.Equal(t, "build", evaluation.FailedChecks[0].Name)
	})

	t.Run("passed when selected checks succeeded", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, failed}, []string{"DCO"}, false)
		assert.Equal(t, waitChecksOutcomePassed, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
		assert.Empty(t, evaluation.FailedChecks)
	})

	t.Run("pending when a selected name is missing", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed}, []string{"DCO", "build"}, false)
		assert.Equal(t, waitChecksOutcomePending, evaluation.Outcome)
		assert.Equal(t, []string{"build"}, evaluation.MissingSelected)
	})

	t.Run("timeout wins over pending selected names", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed}, []string{"DCO", "build"}, true)
		assert.Equal(t, waitChecksOutcomeTimedOut, evaluation.Outcome)
		assert.False(t, evaluation.AllTerminal)
	})

	t.Run("timeout after all checks are terminal", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed}, nil, true)
		assert.Equal(t, waitChecksOutcomeTimedOut, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
	})
}

func Test__NextEvaluateDelay(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	timeoutAt := now.Add(time.Hour)
	quietPeriod := time.Minute
	pollInterval := 5 * time.Minute

	t.Run("returns zero after timeout", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, time.Duration(0), nextEvaluateDelay(timeoutAt, now, timeoutAt, false, quietPeriod, pollInterval))
	})

	t.Run("uses remaining quiet period when all checks are terminal", func(t *testing.T) {
		t.Parallel()
		delay := nextEvaluateDelay(now.Add(10*time.Second), now, timeoutAt, true, quietPeriod, pollInterval)
		assert.Equal(t, 50*time.Second, delay)
	})

	t.Run("returns zero when quiet period elapsed", func(t *testing.T) {
		t.Parallel()
		delay := nextEvaluateDelay(now.Add(quietPeriod), now, timeoutAt, true, quietPeriod, pollInterval)
		assert.Equal(t, time.Duration(0), delay)
	})

	t.Run("uses poll interval while checks are pending", func(t *testing.T) {
		t.Parallel()
		delay := nextEvaluateDelay(now, now, timeoutAt, false, quietPeriod, pollInterval)
		assert.Equal(t, pollInterval, delay)
	})

	t.Run("uses remaining timeout when it is shorter than the poll", func(t *testing.T) {
		t.Parallel()
		delay := nextEvaluateDelay(now, now, now.Add(2*time.Minute), false, quietPeriod, pollInterval)
		assert.Equal(t, 2*time.Minute, delay)
	})
}
