package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logrus.NewEntry(logger)
}

func Test__Jira__Sync(t *testing.T) {
	integration := &Jira{}

	t.Run("no client credentials - setup instructions with callback URL", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{Configuration: map[string]any{}}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Empty(t, integrationContext.BrowserAction.URL)
		assert.Contains(t, integrationContext.BrowserAction.Description, "Atlassian Developer Console")
		assert.Contains(t, integrationContext.BrowserAction.Description, "manage:jira-webhook")
		assert.Contains(t, integrationContext.BrowserAction.Description, "/api/v1/integrations/")
		assert.Contains(t, integrationContext.BrowserAction.Description, "/callback")
	})

	t.Run("missing client secret - setup instructions", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{"clientId": "client-1"},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Empty(t, integrationContext.BrowserAction.URL)
	})

	t.Run("credentials but no access token - authorize button", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Contains(t, integrationContext.BrowserAction.Description, "authorize SuperPlane")

		actionURL, parseErr := url.Parse(integrationContext.BrowserAction.URL)
		require.NoError(t, parseErr)
		assert.Equal(t, "auth.atlassian.com", actionURL.Host)
		assert.Equal(t, "/authorize", actionURL.Path)

		params := actionURL.Query()
		assert.Equal(t, "client-1", params.Get("client_id"))
		assert.Equal(t, "code", params.Get("response_type"))
		assert.Equal(t, scopeList, params.Get("scope"))
		assert.NotEmpty(t, params.Get("state"))

		// The generated state is persisted so the callback can validate it.
		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.State)
		assert.Equal(t, *metadata.State, params.Get("state"))
	})

	t.Run("state is not regenerated on subsequent syncs", func(t *testing.T) {
		state := "existing-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
			Metadata: Metadata{State: &state},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Contains(t, integrationContext.BrowserAction.URL, "state=existing-state")
	})

	t.Run("valid access token - ready + populated projects", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`))},
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			HTTP:        httpContext,
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationContext.State)
		assert.Nil(t, integrationContext.BrowserAction)

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.User)
		assert.Equal(t, "acct-1", metadata.User.AccountID)
		assert.Equal(t, testCloudID, metadata.CloudID)
		require.Len(t, metadata.Projects, 1)
		assert.Equal(t, "TEST", metadata.Projects[0].Key)
	})

	t.Run("credential verification failure marks the integration errored", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`))},
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			HTTP:        httpContext,
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		assert.Equal(t, "error", integrationContext.State)
		assert.Contains(t, integrationContext.StateDescription, "verifying Jira credentials")
	})
}

func Test__Jira__HandleRequest(t *testing.T) {
	integration := &Jira{}

	t.Run("non-callback path -> 404", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Request:     httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/other", nil),
			Response:    recorder,
			Integration: newAuthorizedIntegration(),
			Logger:      newLogger(),
		})

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("valid callback exchanges code, resolves site, stores tokens and becomes ready", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
			Metadata: Metadata{State: &state},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"cb-access","refresh_token":"cb-refresh","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net"}]`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			},
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/callback?code=auth-code&state=expected-state", nil)

		integration.HandleRequest(core.HTTPRequestContext{
			Request:        request,
			Response:       recorder,
			BaseURL:        "https://sp.example.com",
			OrganizationID: "org-1",
			HTTP:           httpContext,
			Integration:    integrationContext,
			Logger:         newLogger(),
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "ready", integrationContext.State)

		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		refreshToken, _ := findSecret(integrationContext, SecretOAuthRefreshToken)
		assert.Equal(t, "cb-access", accessToken)
		assert.Equal(t, "cb-refresh", refreshToken)

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "cloud-1", metadata.CloudID)
		assert.Equal(t, "https://test.atlassian.net", metadata.SiteURL)
		assert.Equal(t, "Test Site", metadata.SiteName)
		assert.Nil(t, metadata.State)
	})

	t.Run("accessible resources failure redirects with an error state", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
			Metadata: Metadata{State: &state},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"cb-access","refresh_token":"cb-refresh","expires_in":3600}`,
				))},
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden"}`))},
			},
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/callback?code=auth-code&state=expected-state", nil)

		integration.HandleRequest(core.HTTPRequestContext{
			Request:        request,
			Response:       recorder,
			BaseURL:        "https://sp.example.com",
			OrganizationID: "org-1",
			HTTP:           httpContext,
			Integration:    integrationContext,
			Logger:         newLogger(),
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "error", integrationContext.State)
		assert.Contains(t, integrationContext.StateDescription, "failed to resolve Jira site")

		// The tokens are still stored, so the next sync can pick up from here.
		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Equal(t, "cb-access", accessToken)
	})

	t.Run("state mismatch does not store tokens", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
			Metadata: Metadata{State: &state},
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/callback?code=auth-code&state=wrong-state", nil)

		integration.HandleRequest(core.HTTPRequestContext{
			Request:        request,
			Response:       recorder,
			BaseURL:        "https://sp.example.com",
			OrganizationID: "org-1",
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationContext,
			Logger:         newLogger(),
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)

		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Empty(t, accessToken)
	})

	t.Run("rejects a missing code", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
		}

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/callback?state=state-1", nil)

		integration.HandleRequest(core.HTTPRequestContext{
			Request:     request,
			Response:    recorder,
			BaseURL:     "https://sp.example.com",
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
	})

	t.Run("missing config -> internal server error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/callback?code=code-1&state=state-1", nil)

		integration.HandleRequest(core.HTTPRequestContext{
			Request:     request,
			Response:    recorder,
			Integration: &contexts.IntegrationContext{},
			Logger:      newLogger(),
		})

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}

func Test__Jira__Definition(t *testing.T) {
	integration := &Jira{}

	assert.Equal(t, "jira", integration.Name())
	assert.Equal(t, "Jira", integration.Label())
	assert.Equal(t, "jira", integration.Icon())

	actions := integration.Actions()
	assert.Len(t, actions, 18)

	triggers := integration.Triggers()
	require.Len(t, triggers, 1)
	assert.Equal(t, "jira.onIssue", triggers[0].Name())
}
