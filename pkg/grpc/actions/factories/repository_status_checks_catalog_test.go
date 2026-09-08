package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggestIntegrationFromCheckURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
		want string
	}{
		{name: "semaphore ci host", url: "https://acme.semaphoreci.com/workflows/abc", want: "semaphore"},
		{name: "semaphore app host", url: "https://me.semaphore.com/jobs/1", want: "semaphore"},
		{name: "circleci host", url: "https://app.circleci.com/pipelines/github/acme/api/12", want: "circleci"},
		{name: "harness host", url: "https://app.harness.io/ng/account/x/ci/pipeline", want: "harness"},
		{name: "github actions stays empty", url: "https://github.com/acme/api/actions/runs/99", want: ""},
		{name: "unknown host stays empty", url: "https://ci.example.com/job/lint", want: ""},
		{name: "blank url stays empty", url: "", want: ""},
		{name: "invalid url stays empty", url: "://not-a-url", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, suggestIntegrationFromCheckURL(tc.url))
		})
	}
}

func TestMergeRepositoryStatusChecks(t *testing.T) {
	t.Parallel()

	t.Run("required checks are selected and listed first", func(t *testing.T) {
		t.Parallel()
		checks := mergeRepositoryStatusChecks(
			[]string{"lint", "unit"},
			[]observedRepositoryStatusCheck{
				{Name: "unit", DetailsURL: "https://github.com/acme/api/actions/runs/1"},
				{Name: "e2e", DetailsURL: "https://app.circleci.com/pipelines/github/acme/api/2"},
			},
		)

		assert.Equal(t, []repositoryStatusCheck{
			{Name: "lint", Required: true},
			{Name: "unit", Required: true, DetailsURL: "https://github.com/acme/api/actions/runs/1"},
			{Name: "e2e", DetailsURL: "https://app.circleci.com/pipelines/github/acme/api/2", SuggestedIntegration: "circleci"},
		}, checks)
	})

	t.Run("duplicate names keep required and the first details url", func(t *testing.T) {
		t.Parallel()
		checks := mergeRepositoryStatusChecks(
			[]string{"CI"},
			[]observedRepositoryStatusCheck{
				{Name: "ci", DetailsURL: "https://acme.semaphoreci.com/workflows/1"},
				{Name: "CI", DetailsURL: "https://app.circleci.com/pipelines/github/acme/api/2"},
			},
		)

		assert.Equal(t, []repositoryStatusCheck{
			{Name: "CI", Required: true, DetailsURL: "https://acme.semaphoreci.com/workflows/1", SuggestedIntegration: "semaphore"},
		}, checks)
	})

	t.Run("blank names are skipped", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, mergeRepositoryStatusChecks([]string{" "}, []observedRepositoryStatusCheck{{Name: ""}}))
	})
}
