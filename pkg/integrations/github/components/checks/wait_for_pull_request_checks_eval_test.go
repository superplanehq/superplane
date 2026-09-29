package checks

import (
	"strings"
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
				Output: &github.CheckRunOutput{
					Title:   github.Ptr("DCO required"),
					Summary: github.Ptr("The DCO check failed.\nSee the log."),
				},
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
				Context:     github.Ptr("ci/semaphore"),
				State:       github.Ptr("success"),
				Description: github.Ptr("CI"),
				TargetURL:   github.Ptr("https://example.com/ci-later"),
			},
		},
	}

	checks := normalizePullRequestChecks(checkRuns, combined)
	require.Len(t, checks, 3)
	assert.Equal(t, "check-run:dco:DCO", checks[0].Key)
	assert.Equal(t, "failure", checks[0].Conclusion)
	assert.Equal(t, "DCO required", checks[0].Description)
	assert.Equal(t, "\n\nThe DCO check failed.\nSee the log.", checks[0].Summary)
	assert.Equal(t, "check-run:github-actions:lint", checks[1].Key)
	assert.Equal(t, checkStatusPending, checks[1].Status)
	assert.Equal(t, "status:ci/semaphore", checks[2].Key)
	assert.Equal(t, "success", checks[2].Conclusion)
	assert.Equal(t, "CI", checks[2].Summary)
	assert.Equal(t, "https://example.com/ci-later", checks[2].DetailsURL)
}

func Test__NormalizePullRequestChecks_KeepsCheckRunHTMLBody(t *testing.T) {
	t.Parallel()

	previewTable := strings.Join([]string{
		"<table>",
		"<tr><td><strong>Preview URL:</strong></td>",
		`<td><a href="https://preview.pages.dev">Visit Preview</a></td></tr>`,
		"</table>",
	}, "\n")
	details := "Last deploy finished in 12s."

	checks := normalizePullRequestChecks(&github.ListCheckRunsResults{
		CheckRuns: []*github.CheckRun{
			{
				Name:       github.Ptr("Cloudflare Pages"),
				Status:     github.Ptr("completed"),
				Conclusion: github.Ptr("success"),
				DetailsURL: github.Ptr("https://example.com/pages"),
				App:        &github.App{Slug: github.Ptr("cloudflare-pages")},
				Output: &github.CheckRunOutput{
					Title:   github.Ptr("Deploy successful"),
					Summary: github.Ptr(previewTable),
					Text:    github.Ptr(details),
				},
			},
		},
	}, nil)

	require.Len(t, checks, 1)
	assert.Equal(t, "success", checks[0].Conclusion)
	assert.Equal(t, "Deploy successful", checks[0].Description)
	assert.Equal(t, "\n\n"+previewTable+"\n\n"+details, checks[0].Summary)
	assert.Contains(t, checks[0].Summary, "https://preview.pages.dev")
}

func Test__NormalizePullRequestChecks_SkipsDuplicateCheckRunText(t *testing.T) {
	t.Parallel()

	body := "<p>Deploy successful.</p>"
	checks := normalizePullRequestChecks(&github.ListCheckRunsResults{
		CheckRuns: []*github.CheckRun{
			{
				Name:   github.Ptr("Cloudflare Pages"),
				Status: github.Ptr("completed"),
				App:    &github.App{Slug: github.Ptr("cloudflare-pages")},
				Output: &github.CheckRunOutput{
					Summary: github.Ptr(body),
					Text:    github.Ptr(body),
				},
			},
		},
	}, nil)

	require.Len(t, checks, 1)
	assert.Equal(t, "\n\n"+body, checks[0].Summary)
}

func Test__NormalizePullRequestChecks_LeavesCommitStatusDescription(t *testing.T) {
	t.Parallel()

	checks := normalizePullRequestChecks(nil, &github.CombinedStatus{
		Statuses: []*github.RepoStatus{
			{
				Context:     github.Ptr("ci/semaphore"),
				State:       github.Ptr("success"),
				Description: github.Ptr("CI passed"),
			},
		},
	})

	require.Len(t, checks, 1)
	assert.Equal(t, "CI passed", checks[0].Summary)
}

func Test__EvaluatePullRequestChecks(t *testing.T) {
	t.Parallel()

	passed := PullRequestCheck{Key: "check-run:dco:DCO", Name: "DCO", Status: checkStatusCompleted, Conclusion: "success"}
	failed := PullRequestCheck{Key: "check-run:ci:build", Name: "build", Status: checkStatusCompleted, Conclusion: "failure"}
	pending := PullRequestCheck{Key: "status:ci", Name: "ci", Status: checkStatusPending}

	t.Run("pending while any selected check is running", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, pending}, []string{"DCO", "ci"}, false)
		assert.Equal(t, waitChecksOutcomePending, evaluation.Outcome)
		assert.False(t, evaluation.AllTerminal)
	})

	t.Run("failed when a selected check failed", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, failed}, []string{"DCO", "build"}, false)
		assert.Equal(t, waitChecksOutcomeFailed, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
		require.Len(t, evaluation.FailedChecks, 1)
		assert.Equal(t, "build", evaluation.FailedChecks[0].Name)
	})

	t.Run("passed when remaining checks were cancelled", func(t *testing.T) {
		t.Parallel()
		cancelled := PullRequestCheck{Key: "check-run:ci:build", Name: "build", Status: checkStatusCompleted, Conclusion: "cancelled"}
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed, cancelled}, []string{"DCO", "build"}, false)
		assert.Equal(t, waitChecksOutcomePassed, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
		assert.Empty(t, evaluation.FailedChecks)
	})

	t.Run("failed when a check failed among cancelled checks", func(t *testing.T) {
		t.Parallel()
		cancelled := PullRequestCheck{Key: "check-run:ci:lint", Name: "lint", Status: checkStatusCompleted, Conclusion: "cancelled"}
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{failed, cancelled}, []string{"build", "lint"}, false)
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

	t.Run("timeout after selected checks are terminal", func(t *testing.T) {
		t.Parallel()
		evaluation := evaluatePullRequestChecks([]PullRequestCheck{passed}, []string{"DCO"}, true)
		assert.Equal(t, waitChecksOutcomeTimedOut, evaluation.Outcome)
		assert.True(t, evaluation.AllTerminal)
	})
}

func Test__NextEvaluateDelay(t *testing.T) {
	t.Parallel()

	now := time.Now()
	timeoutAt := now.Add(time.Hour)
	pollInterval := 5 * time.Minute

	t.Run("returns zero after timeout", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, time.Duration(0), nextEvaluateDelay(timeoutAt, timeoutAt, pollInterval))
	})

	t.Run("uses poll interval while checks are pending", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, pollInterval, nextEvaluateDelay(now, timeoutAt, pollInterval))
	})

	t.Run("uses remaining timeout when it is shorter than the poll", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2*time.Minute, nextEvaluateDelay(now, now.Add(2*time.Minute), pollInterval))
	})
}
