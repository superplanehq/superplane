package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__BuildAuthorizeURL(t *testing.T) {
	url := BuildAuthorizeURL("client-1", "https://superplane.example.com/api/v1/integrations/int-1/redirect", "state-1")

	assert.Contains(t, url, "https://auth.atlassian.com/authorize?")
	assert.Contains(t, url, "client_id=client-1")
	assert.Contains(t, url, "state=state-1")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "audience=api.atlassian.com")
	assert.Contains(t, url, "prompt=consent")
}

func Test__ExchangeCodeForToken(t *testing.T) {
	t.Run("valid exchange returns tokens", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600,"scope":"read:jira-work"}`,
					)),
				},
			},
		}

		token, err := exchangeCodeForToken(httpCtx, "client-1", "secret-1", "code-1", "https://example.com/redirect")
		require.NoError(t, err)
		assert.Equal(t, "access-1", token.AccessToken)
		assert.Equal(t, "refresh-1", token.RefreshToken)
		assert.Equal(t, int64(3600), token.ExpiresIn)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, atlassianTokenURL, req.URL.String())
		body, _ := io.ReadAll(req.Body)
		assert.Contains(t, string(body), `"grant_type":"authorization_code"`)
		assert.Contains(t, string(body), `"code":"code-1"`)
	})

	t.Run("error response is surfaced", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		_, err := exchangeCodeForToken(httpCtx, "client-1", "secret-1", "bad-code", "https://example.com/redirect")
		require.ErrorContains(t, err, "invalid_grant")
	})
}

func Test__RefreshAccessToken(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"access_token":"access-2","refresh_token":"refresh-2","expires_in":3600}`,
				)),
			},
		},
	}

	token, err := refreshAccessToken(httpCtx, "client-1", "secret-1", "refresh-1")
	require.NoError(t, err)
	assert.Equal(t, "access-2", token.AccessToken)
	assert.Equal(t, "refresh-2", token.RefreshToken)

	require.Len(t, httpCtx.Requests, 1)
	body, _ := io.ReadAll(httpCtx.Requests[0].Body)
	assert.Contains(t, string(body), `"grant_type":"refresh_token"`)
	assert.Contains(t, string(body), `"refresh_token":"refresh-1"`)
}

func Test__FetchAccessibleResources(t *testing.T) {
	t.Run("returns accessible sites", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net","scopes":["read:jira-work"]}]`,
					)),
				},
			},
		}

		resources, err := fetchAccessibleResources(httpCtx, "access-1")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "cloud-1", resources[0].ID)
		assert.Equal(t, "https://test.atlassian.net", resources[0].URL)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "Bearer access-1", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("no accessible sites is an error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			},
		}

		_, err := fetchAccessibleResources(httpCtx, "access-1")
		require.ErrorContains(t, err, "no accessible Jira sites")
	})
}
