package gitlab

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AuthService__ExchangeCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"access_token": "access-123",
					"refresh_token": "refresh-123",
					"expires_in": 7200
				}`),
			},
		}

		service := NewAuthService(mock)
		resp, err := service.ExchangeCode("https://gitlab.com", "id", "secret", "code-123", "redirect")
		
		require.NoError(t, err)
		assert.Equal(t, "access-123", resp.AccessToken)
		assert.Equal(t, "refresh-123", resp.RefreshToken)
		assert.Equal(t, 7200, resp.ExpiresIn)
		
		require.Len(t, mock.Requests, 1)
		req := mock.Requests[0]
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitlab.com/oauth/token", req.URL.String())
		
		body, _ := io.ReadAll(req.Body)
		values, _ := url.ParseQuery(string(body))
		assert.Equal(t, "authorization_code", values.Get("grant_type"))
		assert.Equal(t, "code-123", values.Get("code"))
	})

	t.Run("error response", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusBadRequest, `{"error": "invalid_grant"}`),
			},
		}

		service := NewAuthService(mock)
		_, err := service.ExchangeCode("https://gitlab.com", "id", "secret", "code", "redirect")
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 400")
	})
}

func Test__AuthService__RefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"access_token": "access-new",
					"refresh_token": "refresh-new",
					"expires_in": 7200
				}`),
			},
		}

		service := NewAuthService(mock)
		resp, err := service.RefreshToken("https://gitlab.com", "id", "secret", "refresh-old")
		
		require.NoError(t, err)
		assert.Equal(t, "access-new", resp.AccessToken)
		
		require.Len(t, mock.Requests, 1)
		req := mock.Requests[0]
		assert.Equal(t, "POST", req.Method)
		
		body, _ := io.ReadAll(req.Body)
		values, _ := url.ParseQuery(string(body))
		assert.Equal(t, "refresh_token", values.Get("grant_type"))
		assert.Equal(t, "refresh-old", values.Get("refresh_token"))
	})
}

func Test__AuthService__HandleCallback(t *testing.T) {
	// Re-using mock client with default response
	mock := &contexts.HTTPContext{
		Responses: []*http.Response{
			GitlabMockResponse(http.StatusOK, `{"access_token": "ok"}`),
		},
	}
	service := NewAuthService(mock)

	t.Run("valid callback", func(t *testing.T) {
		state := "xyz"
		req, _ := http.NewRequest("GET", "/?code=123&state="+url.QueryEscape(state), nil)
		id := "id"
		secret := "secret"
		config := &Configuration{
			BaseURL: "https://gitlab.com",
			ClientID: &id,
			ClientSecret: &secret,
		}
		
		resp, err := service.HandleCallback(req, config, state, "uri")
		require.NoError(t, err)
		assert.Equal(t, "ok", resp.AccessToken)
	})

	t.Run("invalid state", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/?code=123&state=bad", nil)
		config := &Configuration{}
		
		_, err := service.HandleCallback(req, config, "valid-state", "uri")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})

	t.Run("error param", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/?error=access_denied&error_description=bad", nil)
		config := &Configuration{}
		
		_, err := service.HandleCallback(req, config, "xyz", "uri")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OAuth error")
	})
}
