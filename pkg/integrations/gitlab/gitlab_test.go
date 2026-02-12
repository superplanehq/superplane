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
				"authType":    AuthTypePersonalAccessToken,
				"accessToken": "pat-123",
				"groupId":     "123",
				"baseUrl":     "https://gitlab.com",
			},
		}

		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1}`),
				GitlabMockResponse(http.StatusOK, `[{"id": 1, "path_with_namespace": "group/project1", "web_url": "https://gitlab.com/group/project1"}]`),
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
			HTTP:          mockHTTP,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.State)

		require.Len(t, mockHTTP.Requests, 2)
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockHTTP.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=1", mockHTTP.Requests[1].URL.String())
	})

	t.Run("personal access token - missing token - error state", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"groupId":     "123",
				"accessToken": "",
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "access token is required")
		assert.Empty(t, ctx.State)
		assert.Nil(t, ctx.BrowserAction)
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
		assert.NotNil(t, ctx.BrowserAction)
		assert.Contains(t, ctx.BrowserAction.Description, "Click the **Continue** button to go to the Applications page in GitLab")
		assert.Equal(t, "https://gitlab.com/-/user_settings/applications", ctx.BrowserAction.URL)
	})

	t.Run("oauth - missing client secret - setup instructions", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAppOAuth,
				"groupId":  "123",
				"clientId": "id",
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
		})

		require.NoError(t, err)
		assert.NotNil(t, ctx.BrowserAction)
		assert.Contains(t, ctx.BrowserAction.Description, "Click the **Continue** button to go to the Applications page in GitLab")
		assert.Equal(t, "https://gitlab.com/-/user_settings/applications", ctx.BrowserAction.URL)
	})

	t.Run("oauth - has client id, no tokens - connect button", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":     AuthTypeAppOAuth,
				"groupId":      "123",
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Metadata: Metadata{
				User: &UserMetadata{
					ID:       123,
					Name:     "John Doe",
					Username: "johndoe",
				},
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
		})

		require.NoError(t, err)
		assert.NotNil(t, ctx.BrowserAction)
		assert.Contains(t, ctx.BrowserAction.URL, "/oauth/authorize")
		assert.Contains(t, ctx.BrowserAction.Description, "authorize SuperPlane")

		// Verify metadata preservation
		metadata, ok := ctx.Metadata.(Metadata)
		assert.True(t, ok)
		assert.Equal(t, 123, metadata.User.ID)
		assert.Equal(t, "John Doe", metadata.User.Name)
		assert.Equal(t, "johndoe", metadata.User.Username)
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("oauth - has tokens - success", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":     AuthTypeAppOAuth,
				"groupId":      "123",
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken:  {Name: OAuthAccessToken, Value: []byte("access-token")},
				OAuthRefreshToken: {Name: OAuthRefreshToken, Value: []byte("refresh-token")},
			},
		}

		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"access_token": "new-access-token",
					"refresh_token": "new-refresh-token",
					"expires_in": 7200,
					"token_type": "Bearer"
				}`),
				GitlabMockResponse(http.StatusOK, `{"id": 1, "name": "John Doe", "username": "johndoe"}`),
				GitlabMockResponse(http.StatusOK, `[{"id": 1, "path_with_namespace": "group/project1", "web_url": "https://gitlab.com/group/project1"}]`),
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
			HTTP:          mockHTTP,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.State)

		require.Len(t, mockHTTP.Requests, 3)
		assert.Equal(t, "https://gitlab.com/oauth/token", mockHTTP.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockHTTP.Requests[1].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=1", mockHTTP.Requests[2].URL.String())
	})

	t.Run("oauth - access token present but no refresh token - success", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":     AuthTypeAppOAuth,
				"groupId":      "123",
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("existing-access-token")},
			},
		}

		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				// No token refresh request expected
				GitlabMockResponse(http.StatusOK, `{"id": 1}`),
				GitlabMockResponse(http.StatusOK, `[{"id": 1, "path_with_namespace": "group/project1", "web_url": "https://gitlab.com/group/project1"}]`),
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: ctx.Configuration,
			Integration:   ctx,
			HTTP:          mockHTTP,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", ctx.State)

		// Verification: Should skip token endpoint and go straight to API calls
		require.Len(t, mockHTTP.Requests, 2)
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockHTTP.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=1", mockHTTP.Requests[1].URL.String())
	})

	t.Run("error cases", func(t *testing.T) {
		t.Run("missing authType", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Configuration: map[string]any{},
			}
			err := g.Sync(core.SyncContext{
				Configuration: ctx.Configuration,
				Integration:   ctx,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "authType is required")
		})

		t.Run("unknown authType", func(t *testing.T) {
			ctx := &contexts.IntegrationContext{
				Configuration: map[string]any{"authType": "unknown"},
			}
			err := g.Sync(core.SyncContext{
				Configuration: ctx.Configuration,
				Integration:   ctx,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown authType")
		})
	})
}

func Test__GitLab__HandleRequest(t *testing.T) {
	g := &GitLab{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("handle callback success", func(t *testing.T) {
		state := "xyz"
		ctx := &contexts.IntegrationContext{
			Metadata: Metadata{State: &state},
			Configuration: map[string]any{
				"clientId":     "id",
				"clientSecret": "secret",
				"baseUrl":      "https://gitlab.com",
				"authType":     AuthTypeAppOAuth,
			},
			Secrets: make(map[string]core.IntegrationSecret),
		}

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/callback?code=code123&state="+url.QueryEscape(state), nil)

		// Sequence: Exchange Code -> Verify (User) -> Verify (Projects)
		mockHTTP := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
						"access_token": "access",
						"refresh_token": "refresh",
						"expires_in": 3600
					}`),
				GitlabMockResponse(http.StatusOK, `{"id": 1, "name": "John Doe", "username": "johndoe"}`),
				GitlabMockResponse(http.StatusOK, `[{"id": 1, "path_with_namespace": "group/project1", "web_url": "https://gitlab.com/group/project1"}]`),
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

		require.Len(t, mockHTTP.Requests, 3)
		assert.Equal(t, "https://gitlab.com/oauth/token", mockHTTP.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockHTTP.Requests[1].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=1", mockHTTP.Requests[2].URL.String())

		assert.Equal(t, 1, ctx.Metadata.(Metadata).User.ID)
		assert.Equal(t, "John Doe", ctx.Metadata.(Metadata).User.Name)
		assert.Equal(t, "johndoe", ctx.Metadata.(Metadata).User.Username)
		assert.Len(t, ctx.Metadata.(Metadata).Projects, 1)
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
			state := "valid-state"
			ctx := &contexts.IntegrationContext{
				Metadata: Metadata{State: &state},
				Configuration: map[string]any{
					"clientId":     "id",
					"clientSecret": "secret",
					"baseUrl":      "https://gitlab.com",
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
				HTTP:        mockHTTP,
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
		{"https://gitlab.com/", "https://gitlab.com"},
		{"", "https://gitlab.com"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeBaseURL(tc.input))
		})
	}
}

func gitlabHeaders(event, token string) http.Header {
	headers := http.Header{}
	if event != "" {
		headers.Set("X-Gitlab-Event", event)
	}

	if token != "" {
		headers.Set("X-Gitlab-Token", token)
	}

	return headers
}
