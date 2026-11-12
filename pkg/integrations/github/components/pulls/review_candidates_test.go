package pulls

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func TestReviewCandidatesReadsAllPagesAndFailsIncompleteDiscovery(t *testing.T) {
	for _, failLastPage := range []bool{false, true} {
		t.Run(map[bool]string{false: "all pages", true: "incomplete discovery"}[failLastPage], func(t *testing.T) {
			first := mocks.GitHubResponse(200, `[{"id":1,"login":"author","type":"User","permissions":{"push":true}},{"id":2,"login":"first","type":"User","permissions":{"push":true}}]`)
			first.Header.Set("Link", `<https://api.github.com/repos/testhq/hello/collaborators?page=2>; rel="next"`)
			last := mocks.GitHubResponse(200, `[{"id":3,"login":"last","type":"User","permissions":{"admin":true}},{"id":4,"login":"bot","type":"Bot","permissions":{"push":true}}]`)
			if failLastPage {
				last = mocks.GitHubResponse(503, `{"message":"unavailable"}`)
			}
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mocks.GitHubResponse(200, `{"state":"open","user":{"login":"author"},"head":{"sha":"abc"},"base":{"ref":"main","sha":"base"}}`), first, last}}
			state := &contexts.ExecutionStateContext{}
			err := (&ListPullRequestReviewCandidates{}).Execute(core.ExecutionContext{Configuration: map[string]any{"repository": "hello", "pullNumber": 42}, Integration: mocks.IntegrationContextForNewSetupFlow(), HTTP: httpCtx, ExecutionState: state})
			if failLastPage {
				require.Error(t, err)
				require.Empty(t, state.Payloads)
				return
			}
			require.NoError(t, err)
			require.Len(t, state.Payloads, 1)
			payload := state.Payloads[0].(map[string]any)["data"].(map[string]any)
			require.Equal(t, []ReviewerIdentity{{ID: 2, Login: "first"}, {ID: 3, Login: "last"}}, payload["candidates"])
			require.Equal(t, "open", payload["state"])
		})
	}
}

func TestReviewComponentsSetupRejectsInvalidInputs(t *testing.T) {
	for _, test := range []struct {
		name   string
		action core.Action
		config map[string]any
	}{
		{"missing repository", &ListPullRequestReviewCandidates{}, map[string]any{"pullNumber": 42}},
		{"invalid number", &ListPullRequestReviewCandidates{}, map[string]any{"repository": "hello", "pullNumber": 0}},
		{"empty selection", &EnsurePullRequestReviewers{}, map[string]any{"repository": "hello", "pullNumber": 42, "reviewers": []any{}}},
		{"missing identity", &EnsurePullRequestReviewers{}, map[string]any{"repository": "hello", "pullNumber": 42, "reviewers": []any{map[string]any{"login": "first"}}}},
		{"duplicate selection", &EnsurePullRequestReviewers{}, map[string]any{"repository": "hello", "pullNumber": 42, "reviewers": []any{map[string]any{"id": 2, "login": "first"}, map[string]any{"id": 2, "login": "FIRST"}}}},
	} {
		t.Run(test.name, func(t *testing.T) { require.Error(t, test.action.Setup(core.SetupContext{Configuration: test.config})) })
	}
}
