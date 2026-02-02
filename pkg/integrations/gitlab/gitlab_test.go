package gitlab

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func GitlabMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}


func Test__GitLab__Sync(t *testing.T) {
	g := &GitLab{}

	t.Run("personal access token - success", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":            AuthTypePersonalAccessToken,
				"personalAccessToken": "pat-123",
				"groupId":             "123",
				"baseUrl":             "https://gitlab.com",
			},
		}

		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 123}`),
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
			HTTP:          mockHTTP,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.State)
		
		require.Len(t, mockHTTP.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123", mockHTTP.Requests[0].URL.String())
	})

	t.Run("personal access token - missing token - pending", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":            AuthTypePersonalAccessToken,
				"groupId":             "123",
				"personalAccessToken": "",
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
		})

		require.NoError(t, err)
		assert.Equal(t, "pending", ctx.State)
		assert.NotNil(t, ctx.BrowserAction)
		assert.Contains(t, ctx.BrowserAction.Description, "Personal Access Token Setup")
	})

	t.Run("oauth - missing client id - setup instructions", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAppOAuth,
				"groupId":  "123",
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
		})

		require.NoError(t, err)
		assert.Equal(t, "pending", ctx.State)
		assert.NotNil(t, ctx.BrowserAction)
		assert.Contains(t, ctx.BrowserAction.Description, "Step 1: Create a GitLab OAuth Application")
	})

	t.Run("oauth - has client id, no tokens - connect button", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":     AuthTypeAppOAuth,
				"groupId":      "123",
				"clientId":     "id",
				"clientSecret": "secret",
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
		})

		require.NoError(t, err)
		assert.Equal(t, "pending", ctx.State)
		assert.NotNil(t, ctx.BrowserAction)
		assert.Contains(t, ctx.BrowserAction.URL, "/oauth/authorize")
		assert.Contains(t, ctx.BrowserAction.Description, "Connect to GitLab")
	})

	t.Run("oauth - has tokens - refresh success", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":     AuthTypeAppOAuth,
				"groupId":      "123",
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthRefreshToken: {Name: OAuthRefreshToken, Value: []byte("refresh-token")},
			},
		}

		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
						"access_token": "new-access", 
						"refresh_token": "new-refresh", 
						"expires_in": 3600
					}`),
				GitlabMockResponse(http.StatusOK, `{"id": 123}`),
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
			HTTP:          mockHTTP,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.State)
		
		require.Len(t, mockHTTP.Requests, 2)
		assert.Equal(t, "https://gitlab.com/oauth/token", mockHTTP.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123", mockHTTP.Requests[1].URL.String())

		secrets, _ := ctx.GetSecrets()
		var accessToken string
		for _, s := range secrets {
			if s.Name == OAuthAccessToken {
				accessToken = string(s.Value)
			}
		}
		assert.Equal(t, "new-access", accessToken)
	})
	
	t.Run("error cases", func(t *testing.T) {
		t.Run("missing authType", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Configuration: map[string]any{},
			}
			err := g.Sync(core.SyncContext{Configuration: ctx.Configuration})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "authType is required")
		})

		t.Run("unknown authType", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Configuration: map[string]any{"authType": "unknown"},
			}
			err := g.Sync(core.SyncContext{Configuration: ctx.Configuration})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown authType")
		})


		t.Run("oauth refresh failure", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":     AuthTypeAppOAuth,
					"groupId":      "123",
					"clientId":     "id",
					"clientSecret": "secret",
				},
				Secrets: map[string]core.IntegrationSecret{
					OAuthRefreshToken: {Name: OAuthRefreshToken, Value: []byte("bad-token")},
				},
			}
			mockHTTP := &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusBadRequest, `{"error":"invalid_grant"}`),
				},
			}
			err := g.Sync(core.SyncContext{
				Configuration: ctx.Configuration,
				Integration:   ctx,
				HTTP:          mockHTTP,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "token expired")
		})

		t.Run("oauth verify failure", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":     AuthTypeAppOAuth,
					"groupId":      "123",
					"clientId":     "id",
					"clientSecret": "secret",
				},
				Secrets: map[string]core.IntegrationSecret{
					OAuthRefreshToken: {Name: OAuthRefreshToken, Value: []byte("token")},
				},
			}
			
			// Refresh OK, Verify Fails
			mockHTTP := &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"access_token": "ok"}`),
					GitlabMockResponse(http.StatusForbidden, "{}"),
				},
			}
			
			err := g.Sync(core.SyncContext{
				Configuration: ctx.Configuration,
				Integration:   ctx,
				HTTP:          mockHTTP,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "verify access token")
		})
	})
}

func Test__GitLab__HandleRequest(t *testing.T) {
	g := &GitLab{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("handle callback success", func(t *testing.T) {
		state := "xyz"
		ctx := &contexts.IntegrationContext{
			Metadata: Metadata{State: state},
			Configuration: map[string]any{
				"clientId": "id",
				"clientSecret": "secret",
				"baseUrl": "https://gitlab.com",
				"authType": AuthTypeAppOAuth,
			},
			Secrets: make(map[string]core.IntegrationSecret),
		}

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/callback?code=code123&state="+url.QueryEscape(state), nil)

		// Sequence: Exchange Code -> Verify
		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
						"access_token": "access",
						"refresh_token": "refresh",
						"expires_in": 3600
					}`),
				GitlabMockResponse(http.StatusOK, `{"id": 123}`),
			},
		}
		
		ctx.Configuration["groupId"] = "123"

		g.HandleRequest(core.HTTPRequestContext{
			Request:     req,
			Response:    recorder,
			Integration: ctx,
			HTTP:        mockHTTP,
			Logger:      logger,
		})
		
		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "ready", ctx.State)
		
		require.Len(t, mockHTTP.Requests, 2)
		assert.Equal(t, "https://gitlab.com/oauth/token", mockHTTP.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123", mockHTTP.Requests[1].URL.String())
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("unknown path", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/unknown", nil)
			
			g.HandleRequest(core.HTTPRequestContext{
				Request:     req,
				Response:    recorder,
				Integration: ctx,
				Logger:      logger,
			})
			
			assert.Equal(t, http.StatusNotFound, recorder.Code)
		})

		t.Run("callback failure", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Metadata: Metadata{State: "valid-state"},
				Configuration: map[string]any{
					"clientId": "id",
					"clientSecret": "secret",
					"baseUrl": "https://gitlab.com",
				},
				Secrets: make(map[string]core.IntegrationSecret),
			}
			
			mockHTTP := &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusBadRequest, "{}"),
				},
			}
			
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/callback?code=bad&state=valid-state", nil)
			
			g.HandleRequest(core.HTTPRequestContext{
				Request:     req,
				Response:    recorder,
				Integration: ctx, 
				HTTP:        mockHTTP, // Use global mock
				Logger:      logger,
			})
			
			assert.Equal(t, http.StatusSeeOther, recorder.Code)
			
			assert.NotContains(t, ctx.State, "error")
		})
	})
}

func Test__GitLab__BaseURLNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gitlab.com", "https://gitlab.com"},
		{"http://gitlab.com", "http://gitlab.com"},
		{"https://gitlab.com", "https://gitlab.com"},
		{"", "https://gitlab.com"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeBaseURL(tc.input))
		})
	}
}
