package linear

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func linearMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func Test__Auth__exchangeCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusOK, `{
					"access_token": "access-123",
					"refresh_token": "refresh-123",
					"expires_in": 86399
				}`),
			},
		}

		auth := NewAuth(mock)
		resp, err := auth.exchangeCode("client-id", "client-secret", "code-123", "https://example.com/callback")

		require.NoError(t, err)
		assert.Equal(t, "access-123", resp.AccessToken)
		assert.Equal(t, "refresh-123", resp.RefreshToken)
		assert.Equal(t, 86399, resp.ExpiresIn)

		require.Len(t, mock.Requests, 1)
		req := mock.Requests[0]
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, linearTokenURL, req.URL.String())

		body, _ := io.ReadAll(req.Body)
		values, _ := url.ParseQuery(string(body))
		assert.Equal(t, "authorization_code", values.Get("grant_type"))
		assert.Equal(t, "code-123", values.Get("code"))
		assert.Equal(t, "client-id", values.Get("client_id"))
		assert.Equal(t, "client-secret", values.Get("client_secret"))
		assert.Equal(t, "https://example.com/callback", values.Get("redirect_uri"))
	})

	t.Run("error response", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusBadRequest, `{"error": "invalid_grant"}`),
			},
		}

		auth := NewAuth(mock)
		_, err := auth.exchangeCode("id", "secret", "code", "redirect")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 400")
	})
}

func Test__Auth__RefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusOK, `{
					"access_token": "access-new",
					"refresh_token": "refresh-new",
					"expires_in": 86399
				}`),
			},
		}

		auth := NewAuth(mock)
		resp, err := auth.RefreshToken("client-id", "client-secret", "refresh-old")

		require.NoError(t, err)
		assert.Equal(t, "access-new", resp.AccessToken)
		assert.Equal(t, "refresh-new", resp.RefreshToken)

		require.Len(t, mock.Requests, 1)
		req := mock.Requests[0]
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, linearTokenURL, req.URL.String())

		body, _ := io.ReadAll(req.Body)
		values, _ := url.ParseQuery(string(body))
		assert.Equal(t, "refresh_token", values.Get("grant_type"))
		assert.Equal(t, "refresh-old", values.Get("refresh_token"))
		assert.Equal(t, "client-id", values.Get("client_id"))
		assert.Equal(t, "client-secret", values.Get("client_secret"))
	})

	t.Run("error response", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusUnauthorized, `{}`),
			},
		}

		auth := NewAuth(mock)
		_, err := auth.RefreshToken("id", "secret", "bad-refresh")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 401")
	})
}

func Test__Auth__HandleCallback(t *testing.T) {
	t.Run("valid callback", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusOK, `{"access_token": "ok", "refresh_token": "ref"}`),
			},
		}

		auth := NewAuth(mock)
		req, _ := http.NewRequest("GET", "/?code=123&state=xyz", nil)
		resp, err := auth.HandleCallback(req, "id", "secret", "xyz", "https://example.com/callback")

		require.NoError(t, err)
		assert.Equal(t, "ok", resp.AccessToken)
		assert.Equal(t, "ref", resp.RefreshToken)
	})

	t.Run("invalid state", func(t *testing.T) {
		auth := NewAuth(&contexts.HTTPContext{})
		req, _ := http.NewRequest("GET", "/?code=123&state=bad", nil)
		_, err := auth.HandleCallback(req, "id", "secret", "valid-state", "uri")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})

	t.Run("missing code", func(t *testing.T) {
		auth := NewAuth(&contexts.HTTPContext{})
		req, _ := http.NewRequest("GET", "/?state=xyz", nil)
		_, err := auth.HandleCallback(req, "id", "secret", "xyz", "uri")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing code or state")
	})

	t.Run("OAuth error param", func(t *testing.T) {
		auth := NewAuth(&contexts.HTTPContext{})
		req, _ := http.NewRequest("GET", "/?error=access_denied&error_description=user+denied", nil)
		_, err := auth.HandleCallback(req, "id", "secret", "xyz", "uri")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "OAuth error")
		assert.Contains(t, err.Error(), "access_denied")
	})
}

func Test__TokenResponse__GetExpiration(t *testing.T) {
	t.Run("returns half of expires_in", func(t *testing.T) {
		resp := TokenResponse{ExpiresIn: 86400}
		assert.Equal(t, 43200, int(resp.GetExpiration().Seconds()))
	})

	t.Run("minimum 1 second", func(t *testing.T) {
		resp := TokenResponse{ExpiresIn: 1}
		assert.Equal(t, 1, int(resp.GetExpiration().Seconds()))
	})

	t.Run("defaults to 1 hour when zero", func(t *testing.T) {
		resp := TokenResponse{ExpiresIn: 0}
		assert.Equal(t, 3600, int(resp.GetExpiration().Seconds()))
	})
}
