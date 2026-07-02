package runcodeagent

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__resolvePullRequest(t *testing.T) {
	t.Run("same-repo open PR", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{resp(`{
			"number":42,"state":"open","html_url":"https://github.com/o/r/pull/42",
			"head":{"ref":"feature","repo":{"full_name":"o/r"}},
			"base":{"ref":"main","repo":{"full_name":"o/r"}}
		}`)}}
		pr, err := resolvePullRequest(httpCtx, "https://github.com/o/r/pull/42", "tok")
		require.NoError(t, err)
		assert.Equal(t, "feature", pr.HeadRef)
		assert.Equal(t, "main", pr.BaseRef)
		assert.Equal(t, "o/r", pr.BaseRepo)
		assert.False(t, pr.isFork())
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Bearer tok", httpCtx.Requests[0].Header.Get("Authorization"))
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/repos/o/r/pulls/42")
	})

	t.Run("fork PR detected", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{resp(`{
			"state":"open","head":{"ref":"feature","repo":{"full_name":"fork/r"}},
			"base":{"ref":"main","repo":{"full_name":"o/r"}}
		}`)}}
		pr, err := resolvePullRequest(httpCtx, "https://github.com/o/r/pull/1", "tok")
		require.NoError(t, err)
		assert.True(t, pr.isFork())
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, err := resolvePullRequest(&contexts.HTTPContext{}, "https://github.com/o/r/issues/1", "tok")
		require.Error(t, err)
	})

	t.Run("API error propagated", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: 404, Body: io.NopCloser(strings.NewReader(`{"message":"Not Found"}`)),
		}}}
		_, err := resolvePullRequest(httpCtx, "https://github.com/o/r/pull/9", "tok")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func Test__resolveGitHubUser(t *testing.T) {
	t.Run("uses name and email when present", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			resp(`{"login":"octocat","id":583231,"name":"The Octocat","email":"octo@github.com"}`),
		}}
		name, email, err := resolveGitHubUser(httpCtx, "tok")
		require.NoError(t, err)
		assert.Equal(t, "The Octocat", name)
		assert.Equal(t, "octo@github.com", email)
	})

	t.Run("falls back to login and noreply email", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			resp(`{"login":"octocat","id":583231,"name":"","email":""}`),
		}}
		name, email, err := resolveGitHubUser(httpCtx, "tok")
		require.NoError(t, err)
		assert.Equal(t, "octocat", name)
		assert.Equal(t, "583231+octocat@users.noreply.github.com", email)
	})

	t.Run("error on failure", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{{
			StatusCode: 401, Body: io.NopCloser(strings.NewReader(`{"message":"Bad credentials"}`)),
		}}}
		_, _, err := resolveGitHubUser(httpCtx, "tok")
		require.Error(t, err)
	})
}
