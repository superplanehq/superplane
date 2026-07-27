package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Auth__ExchangeCode(t *testing.T) {
	t.Run("sends the authorization code grant as JSON", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"at","refresh_token":"rt","expires_in":3600}`,
				))},
			},
		}

		auth := NewAuth(httpContext)
		token, err := auth.ExchangeCode("client-1", "secret-1", "the-code", "https://sp.example.com/callback")
		require.NoError(t, err)
		assert.Equal(t, "at", token.AccessToken)
		assert.Equal(t, "rt", token.RefreshToken)
		assert.Equal(t, 3600, token.ExpiresIn)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, TokenURL, request.URL.String())
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)
		assert.Contains(t, string(body), `"grant_type":"authorization_code"`)
		assert.Contains(t, string(body), `"code":"the-code"`)
	})

	t.Run("non-2xx response -> error with body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		auth := NewAuth(httpContext)
		_, err := auth.ExchangeCode("client-1", "secret-1", "bad-code", "https://sp.example.com/callback")
		require.ErrorContains(t, err, "invalid_grant")
	})

	t.Run("response without access token -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}},
		}

		auth := NewAuth(httpContext)
		_, err := auth.ExchangeCode("client-1", "secret-1", "the-code", "https://sp.example.com/callback")
		require.ErrorContains(t, err, "missing access_token")
	})
}

func Test__Auth__RefreshToken(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"access_token":"new-at","refresh_token":"new-rt","expires_in":3600}`,
			))},
		},
	}

	auth := NewAuth(httpContext)
	token, err := auth.RefreshToken("client-1", "secret-1", "old-rt")
	require.NoError(t, err)
	assert.Equal(t, "new-at", token.AccessToken)
	assert.Equal(t, "new-rt", token.RefreshToken)

	require.Len(t, httpContext.Requests, 1)
	body, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(body), `"grant_type":"refresh_token"`)
	assert.Contains(t, string(body), `"refresh_token":"old-rt"`)
}

func Test__Auth__HandleCallback(t *testing.T) {
	auth := NewAuth(&contexts.HTTPContext{})

	t.Run("provider error is surfaced", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied&error_description=denied", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.ErrorContains(t, err, "access_denied")
	})

	t.Run("missing code -> error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?state=state", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.ErrorContains(t, err, "missing code or state")
	})

	t.Run("state mismatch -> error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=other", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.ErrorContains(t, err, "invalid state")
	})

	// An integration whose state was never generated must reject every callback,
	// even one carrying an attacker-supplied non-empty state.
	t.Run("empty expected state never matches", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=attacker-state", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "", "https://cb")
		require.ErrorContains(t, err, "invalid state")
	})

	t.Run("valid callback exchanges the code", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"at","expires_in":3600}`))},
			},
		}

		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=state", nil)

		token, err := NewAuth(httpContext).HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.NoError(t, err)
		assert.Equal(t, "at", token.AccessToken)
	})
}

func Test__Auth__AccessibleResources(t *testing.T) {
	t.Run("returns accessible sites", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net","scopes":["read:jira-work"]}]`,
				))},
			},
		}

		resources, err := NewAuth(httpContext).AccessibleResources("access-1")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "cloud-1", resources[0].ID)
		assert.Equal(t, "https://test.atlassian.net", resources[0].URL)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "Bearer access-1", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("no accessible sites is an error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))}},
		}

		_, err := NewAuth(httpContext).AccessibleResources("access-1")
		require.ErrorContains(t, err, "no accessible Jira sites")
	})
}

func Test__TokenResponse__GetExpiration(t *testing.T) {
	t.Run("half the token lifetime", func(t *testing.T) {
		response := TokenResponse{ExpiresIn: 3600}
		assert.Equal(t, 1800, int(response.GetExpiration().Seconds()))
	})

	t.Run("defaults to 30 minutes when missing", func(t *testing.T) {
		response := TokenResponse{}
		assert.Equal(t, 1800, int(response.GetExpiration().Seconds()))
	})
}
