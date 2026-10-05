package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func bitbucketResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func workspaceTokenIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{"token": "workspace-token"},
		Metadata:      Metadata{AuthType: AuthTypeWorkspaceAccessToken},
	}
}

func requestJSON(t *testing.T, request *http.Request) map[string]any {
	t.Helper()
	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	decoded := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &decoded))
	return decoded
}

func Test__FindPullRequest__Execute(t *testing.T) {
	component := &FindPullRequest{}

	t.Run("emits the first open pull request on the found channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values":[{"id":42,"title":"feat: Retry"}]}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "acme/widgets", "head": "feat/retry", "base": "main"},
			HTTP:           httpCtx,
			Integration:    workspaceTokenIntegration(),
			ExecutionState: state,
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests", request.URL.Path)
		assert.Equal(t, `source.branch.name="feat/retry" AND destination.branch.name="main"`, request.URL.Query().Get("q"))
		assert.Equal(t, []string{"OPEN"}, request.URL.Query()["state"])
		assert.Equal(t, "Bearer workspace-token", request.Header.Get("Authorization"))

		assert.Equal(t, FindPullRequestFoundChannel, state.Channel)
		assert.Equal(t, PayloadTypePullRequest, state.Type)
	})

	t.Run("emits on the notFound channel when no pull request matches", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"repository": "acme/widgets", "head": "feat/retry"},
			HTTP: &contexts.HTTPContext{Responses: []*http.Response{
				bitbucketResponse(http.StatusOK, `{"values":[]}`),
			}},
			Integration:    workspaceTokenIntegration(),
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.Equal(t, FindPullRequestNotFoundChannel, state.Channel)
	})

	t.Run("rejects a repository without a workspace", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "widgets", "head": "feat/retry"},
			HTTP:           &contexts.HTTPContext{},
			Integration:    workspaceTokenIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})
		require.ErrorContains(t, err, "workspace/repository format")
	})
}

func Test__CreatePullRequest__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		bitbucketResponse(http.StatusCreated, `{"id":42,"links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/42"}}}`),
	}}
	state := &contexts.ExecutionStateContext{}

	err := (&CreatePullRequest{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"repository": "acme/widgets",
			"head":       "feat/retry",
			"base":       "main",
			"title":      "feat: Retry",
			"body":       "Adds retry handling.",
		},
		HTTP:           httpCtx,
		Integration:    workspaceTokenIntegration(),
		ExecutionState: state,
	})
	require.NoError(t, err)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	assert.Equal(t, map[string]any{
		"title":       "feat: Retry",
		"description": "Adds retry handling.",
		"source":      map[string]any{"branch": map[string]any{"name": "feat/retry"}},
		"destination": map[string]any{"branch": map[string]any{"name": "main"}},
		"draft":       false,
	}, requestJSON(t, httpCtx.Requests[0]))
	assert.Equal(t, PayloadTypePullRequest, state.Type)
}

func Test__UpdatePullRequest__Execute(t *testing.T) {
	t.Run("sends only the fields that are set", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"id":42}`),
		}}

		err := (&UpdatePullRequest{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "acme/widgets", "pullNumber": "42", "title": "feat: Retry v2"},
			HTTP:           httpCtx,
			Integration:    workspaceTokenIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})
		require.NoError(t, err)

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42", request.URL.Path)
		assert.Equal(t, map[string]any{"title": "feat: Retry v2"}, requestJSON(t, request))
	})

	t.Run("rejects a pull request ID that is not a number", func(t *testing.T) {
		err := (&UpdatePullRequest{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "acme/widgets", "pullNumber": "abc", "title": "x"},
			HTTP:           &contexts.HTTPContext{},
			Integration:    workspaceTokenIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})
		require.ErrorContains(t, err, "positive number")
	})
}

func Test__Bitbucket__ListResources__DefaultBranch(t *testing.T) {
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		bitbucketResponse(http.StatusOK, `{"full_name":"acme/widgets","mainbranch":{"name":"develop"}}`),
	}}

	resources, err := (&Bitbucket{}).ListResources("default_branch", core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: workspaceTokenIntegration(),
		Parameters:  map[string]string{"repository": "acme/widgets"},
	})
	require.NoError(t, err)

	assert.Equal(t, "/2.0/repositories/acme/widgets", httpCtx.Requests[0].URL.Path)
	require.Len(t, resources, 1)
	assert.Equal(t, "develop", resources[0].Name)
}

func Test__CreatePullRequestComment__Execute(t *testing.T) {
	t.Run("posts the raw comment body", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusCreated, `{"id":7001}`),
		}}

		err := (&CreatePullRequestComment{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "acme/widgets", "pullNumber": "42", "body": "## Visual evidence"},
			HTTP:           httpCtx,
			Integration:    workspaceTokenIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})
		require.NoError(t, err)

		request := httpCtx.Requests[0]
		assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42/comments", request.URL.Path)
		assert.Equal(t, map[string]any{"content": map[string]any{"raw": "## Visual evidence"}}, requestJSON(t, request))
	})

	t.Run("returns the Bitbucket error message", func(t *testing.T) {
		err := (&CreatePullRequestComment{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{"repository": "acme/widgets", "pullNumber": "42", "body": "hi"},
			HTTP: &contexts.HTTPContext{Responses: []*http.Response{
				bitbucketResponse(http.StatusForbidden, `{"type":"error","error":{"message":"Access denied"}}`),
			}},
			Integration:    workspaceTokenIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})
		require.ErrorContains(t, err, "Access denied")
	})
}
