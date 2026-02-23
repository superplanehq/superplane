package linear

import (
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

func Test__Linear__Sync(t *testing.T) {
	integration := &Linear{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("missing clientId and clientSecret -> setup instructions", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{},
			Integration:   appCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.URL, "linear.app/settings/api/applications")
		assert.Contains(t, appCtx.BrowserAction.Description, "OAuth2 Applications")
	})

	t.Run("has clientId but no clientSecret -> setup instructions", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId": "id",
			},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.URL, "linear.app/settings/api/applications")
	})

	t.Run("has credentials but no access token -> authorize button", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "test-client-id",
				"clientSecret": "test-client-secret",
			},
			Secrets: map[string]core.IntegrationSecret{},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		assert.Contains(t, appCtx.BrowserAction.URL, "linear.app/oauth/authorize")
		assert.Contains(t, appCtx.BrowserAction.URL, "client_id=test-client-id")
		assert.Contains(t, appCtx.BrowserAction.URL, "actor=app")
		assert.Contains(t, appCtx.BrowserAction.Description, "authorize SuperPlane")
	})

	t.Run("has tokens -> refreshes and reaches ready", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken:  {Name: OAuthAccessToken, Value: []byte("access-token")},
				OAuthRefreshToken: {Name: OAuthRefreshToken, Value: []byte("refresh-token")},
			},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Token refresh
				linearMockResponse(http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":86399}`),
				// GetViewer
				linearMockResponse(http.StatusOK, `{"data":{"viewer":{"id":"u1","name":"User","email":"u@x.com"}}}`),
				// ListTeams
				linearMockResponse(http.StatusOK, `{"data":{"teams":{"nodes":[{"id":"t1","name":"Team 1","key":"T1"}]}}}`),
				// ListLabels
				linearMockResponse(http.StatusOK, `{"data":{"organization":{"labels":{"nodes":[{"id":"l1","name":"Bug"}]}}}}`),
			},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
			HTTP:          httpCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Nil(t, appCtx.BrowserAction)
	})

	t.Run("has access token but no refresh token -> skips refresh and reaches ready", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
			},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetViewer
				linearMockResponse(http.StatusOK, `{"data":{"viewer":{"id":"u1","name":"User","email":"u@x.com"}}}`),
				// ListTeams
				linearMockResponse(http.StatusOK, `{"data":{"teams":{"nodes":[{"id":"t1","name":"Team 1","key":"T1"}]}}}`),
				// ListLabels
				linearMockResponse(http.StatusOK, `{"data":{"organization":{"labels":{"nodes":[{"id":"l1","name":"Bug"}]}}}}`),
			},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
			HTTP:          httpCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})
}

func Test__Linear__HandleRequest(t *testing.T) {
	l := &Linear{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("unknown path -> 404", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/unknown", nil)

		l.HandleRequest(core.HTTPRequestContext{
			Request:     req,
			Response:    recorder,
			Integration: appCtx,
			Logger:      logger,
		})

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("callback success -> stores tokens and redirects", func(t *testing.T) {
		state := "test-state"
		appCtx := &contexts.IntegrationContext{
			Metadata: Metadata{State: &state},
			Configuration: map[string]any{
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: make(map[string]core.IntegrationSecret),
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Token exchange
				linearMockResponse(http.StatusOK, `{"access_token":"access","refresh_token":"refresh","expires_in":86399}`),
				// GetViewer
				linearMockResponse(http.StatusOK, `{"data":{"viewer":{"id":"u1","name":"User","email":"u@x.com"}}}`),
				// ListTeams
				linearMockResponse(http.StatusOK, `{"data":{"teams":{"nodes":[{"id":"t1","name":"Eng","key":"ENG"}]}}}`),
				// ListLabels
				linearMockResponse(http.StatusOK, `{"data":{"organization":{"labels":{"nodes":[{"id":"l1","name":"Bug"}]}}}}`),
			},
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/callback?code=code123&state="+url.QueryEscape(state), nil)

		l.HandleRequest(core.HTTPRequestContext{
			Request:     req,
			Response:    recorder,
			Integration: appCtx,
			HTTP:        httpCtx,
			Logger:      logger,
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "ready", appCtx.State)
		assert.Equal(t, "access", string(appCtx.Secrets[OAuthAccessToken].Value))
		assert.Equal(t, "refresh", string(appCtx.Secrets[OAuthRefreshToken].Value))
	})

	t.Run("callback failure -> redirect back", func(t *testing.T) {
		state := "valid-state"
		appCtx := &contexts.IntegrationContext{
			Metadata: Metadata{State: &state},
			Configuration: map[string]any{
				"clientId":     "id",
				"clientSecret": "secret",
			},
			Secrets: make(map[string]core.IntegrationSecret),
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusBadRequest, `{"error":"invalid_grant"}`),
			},
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/callback?code=bad&state=valid-state", nil)

		l.HandleRequest(core.HTTPRequestContext{
			Request:     req,
			Response:    recorder,
			Integration: appCtx,
			HTTP:        httpCtx,
			Logger:      logger,
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.NotEqual(t, "ready", appCtx.State)
	})
}
