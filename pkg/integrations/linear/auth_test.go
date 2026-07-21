package linear

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Auth__ExchangeCode(t *testing.T) {
	t.Run("sends the authorization code grant form-encoded", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"access_token":"at","refresh_token":"rt","token_type":"Bearer","expires_in":86399}`),
			},
		}

		auth := NewAuth(httpContext)
		tokenResponse, err := auth.ExchangeCode(testClientID, testClientSecret, "the-code", "https://sp.example.com/callback")
		require.NoError(t, err)
		assert.Equal(t, "at", tokenResponse.AccessToken)
		assert.Equal(t, "rt", tokenResponse.RefreshToken)
		assert.Equal(t, 86399, tokenResponse.ExpiresIn)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, TokenURL, request.URL.String())
		assert.Equal(t, "application/x-www-form-urlencoded", request.Header.Get("Content-Type"))

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)
		form, parseErr := url.ParseQuery(string(body))
		require.NoError(t, parseErr)
		assert.Equal(t, "authorization_code", form.Get("grant_type"))
		assert.Equal(t, "the-code", form.Get("code"))
		assert.Equal(t, testClientID, form.Get("client_id"))
		assert.Equal(t, testClientSecret, form.Get("client_secret"))
		assert.Equal(t, "https://sp.example.com/callback", form.Get("redirect_uri"))
	})

	t.Run("non-200 response -> error with body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		auth := NewAuth(httpContext)
		_, err := auth.ExchangeCode(testClientID, testClientSecret, "bad-code", "https://sp.example.com/callback")
		require.ErrorContains(t, err, "invalid_grant")
	})

	t.Run("response without access token -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(`{}`)},
		}

		auth := NewAuth(httpContext)
		_, err := auth.ExchangeCode(testClientID, testClientSecret, "the-code", "https://sp.example.com/callback")
		require.ErrorContains(t, err, "no access token")
	})
}

func Test__Auth__RefreshToken(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"access_token":"new-at","refresh_token":"new-rt","token_type":"Bearer","expires_in":86399}`),
		},
	}

	auth := NewAuth(httpContext)
	tokenResponse, err := auth.RefreshToken(testClientID, testClientSecret, "old-rt")
	require.NoError(t, err)
	assert.Equal(t, "new-at", tokenResponse.AccessToken)
	assert.Equal(t, "new-rt", tokenResponse.RefreshToken)

	require.Len(t, httpContext.Requests, 1)
	body, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)
	form, parseErr := url.ParseQuery(string(body))
	require.NoError(t, parseErr)
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "old-rt", form.Get("refresh_token"))
}

func Test__Auth__HandleCallback(t *testing.T) {
	auth := NewAuth(&contexts.HTTPContext{})

	t.Run("provider error is surfaced", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied&error_description=denied", nil)

		_, err := auth.HandleCallback(request, testClientID, testClientSecret, "state", "https://cb")
		require.ErrorContains(t, err, "access_denied")
	})

	t.Run("missing code -> error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?state=state", nil)

		_, err := auth.HandleCallback(request, testClientID, testClientSecret, "state", "https://cb")
		require.ErrorContains(t, err, "missing code or state")
	})

	t.Run("state mismatch -> error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=other", nil)

		_, err := auth.HandleCallback(request, testClientID, testClientSecret, "state", "https://cb")
		require.ErrorContains(t, err, "invalid state")
	})

	// An integration whose state was never generated must reject every callback,
	// even one carrying an attacker-supplied non-empty state.
	t.Run("empty expected state never matches", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=attacker-state", nil)

		_, err := auth.HandleCallback(request, testClientID, testClientSecret, "", "https://cb")
		require.ErrorContains(t, err, "invalid state")
	})

	t.Run("valid callback exchanges the code", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"access_token":"at","token_type":"Bearer","expires_in":86399}`),
			},
		}

		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=state", nil)

		tokenResponse, err := NewAuth(httpContext).HandleCallback(request, testClientID, testClientSecret, "state", "https://cb")
		require.NoError(t, err)
		assert.Equal(t, "at", tokenResponse.AccessToken)
	})
}

func Test__TokenResponse__GetExpiration(t *testing.T) {
	t.Run("half the token lifetime", func(t *testing.T) {
		response := TokenResponse{ExpiresIn: 86399}
		assert.Equal(t, 43199, int(response.GetExpiration().Seconds()))
	})

	t.Run("defaults to an hour when missing", func(t *testing.T) {
		response := TokenResponse{}
		assert.Equal(t, 3600, int(response.GetExpiration().Seconds()))
	})
}
