package jira

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

// raceSimulatingHTTPContext simulates a concurrent caller rotating the stored access token while
// this client's own token request is in flight, by mutating the integration's stored secret the
// moment a request reaches the token endpoint - regardless of what response comes back for it.
type raceSimulatingHTTPContext struct {
	integration *contexts.IntegrationContext
	responses   []*http.Response
}

func (h *raceSimulatingHTTPContext) Do(req *http.Request) (*http.Response, error) {
	if req.URL.String() == TokenURL {
		h.integration.CurrentSecrets[SecretOAuthAccessToken] = core.IntegrationSecret{
			Name: SecretOAuthAccessToken, Value: []byte("winner-access"),
		}
	}

	if len(h.responses) == 0 {
		return nil, errors.New("no response mocked")
	}
	resp := h.responses[0]
	h.responses = h.responses[1:]
	return resp, nil
}

func Test__Jira__Sync(t *testing.T) {
	integration := &Jira{}

	t.Run("no client credentials - setup instructions with callback URL", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{Configuration: map[string]any{}}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Empty(t, integrationContext.BrowserAction.URL)
		assert.Contains(t, integrationContext.BrowserAction.Description, "Atlassian Developer Console")
		assert.Contains(t, integrationContext.BrowserAction.Description, "manage:jira-webhook")
		assert.Contains(t, integrationContext.BrowserAction.Description, "/api/v1/integrations/")
		assert.Contains(t, integrationContext.BrowserAction.Description, "/callback")

		// Regression test: most Jira Cloud sites don't have the JSM Ops product (incidents,
		// alerts, heartbeats) enabled at all, and Atlassian's authorize page rejects the whole
		// request outright if the app requests scopes for a product it was never configured
		// with - so by default (ops features off), instructions must not mention them.
		description := integrationContext.BrowserAction.Description
		assert.NotContains(t, description, "Jira Service Management Incident API")
		assert.NotContains(t, description, "Jira Service Management Ops API")
		assert.NotContains(t, description, "incident:jira-service-management")
		assert.NotContains(t, description, "ops-alert:jira-service-management")
		assert.NotContains(t, description, "ops-config:jira-service-management")
	})

	t.Run("no client credentials with ops features enabled - setup instructions include the extra APIs", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{"enableOpsFeatures": true},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)

		// Regression test: incident and ops-alert/ops-config scopes live under separate API
		// products in the Developer Console from the (classic) "Jira Service Management API",
		// which only ever offers servicedesk-request/servicedesk-customer/insight-object scopes -
		// so each API product must be named and paired with only the scopes that belong to it.
		description := integrationContext.BrowserAction.Description
		assert.Contains(t, description, "Jira Service Management Incident API")
		assert.Contains(t, description, "Jira Service Management Ops API")
		assert.Contains(t, description, "read:incident:jira-service-management")
		assert.Contains(t, description, "read:ops-alert:jira-service-management")
		assert.Contains(t, description, "read:ops-config:jira-service-management")
		jsmAPIIndex := strings.Index(description, "**Jira Service Management API**")
		incidentAPIIndex := strings.Index(description, "**Jira Service Management Incident API**")
		require.NotEqual(t, -1, jsmAPIIndex)
		require.NotEqual(t, -1, incidentAPIIndex)
		assert.NotContains(t, description[jsmAPIIndex:incidentAPIIndex], "incident:jira-service-management",
			"incident scopes must not be listed under the plain Jira Service Management API")
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

	// Regression test: an install still carrying config from the old Basic Auth flow (before this
	// branch replaced it with OAuth) must not silently stay "ready" while every call fails. Sync
	// returns a real error here (rather than calling Integration.Error itself) so the existing,
	// unmodified caller (worker/update_integration/create_integration) marks the integration
	// accordingly - the same convention every other Sync failure in this integration already uses.
	t.Run("legacy Basic Auth config with no OAuth credentials errors", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{"siteUrl": "https://old.atlassian.net", "email": "a@b.com", "apiToken": "tok"},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        newLogger(),
		})

		require.ErrorContains(t, err, "no longer supported")
		require.NotNil(t, integrationContext.BrowserAction)
	})

	t.Run("brand new install with no config is not marked errored", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{Configuration: map[string]any{}}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        newLogger(),
		})

		require.NoError(t, err)
		assert.Empty(t, integrationContext.State)
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
		// Ops features are off by default - only the scopes every Jira Cloud site can grant.
		assert.Equal(t, coreScopeList, params.Get("scope"))
		assert.NotEmpty(t, params.Get("state"))

		// The generated state is persisted so the callback can validate it.
		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.State)
		assert.Equal(t, *metadata.State, params.Get("state"))
	})

	// Regression test: requesting JSM Ops scopes (incidents, alerts, heartbeats) from an
	// Atlassian app that was never configured with the matching API products makes Atlassian's
	// authorize page reject the whole request outright - most Jira Cloud sites don't have JSM
	// Ops at all, so those scopes are opt-in via the enableOpsFeatures config option.
	t.Run("credentials with ops features enabled - authorize URL includes ops scopes", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":          "client-1",
				"clientSecret":      "secret-1",
				"enableOpsFeatures": true,
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        newLogger(),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)

		actionURL, parseErr := url.Parse(integrationContext.BrowserAction.URL)
		require.NoError(t, parseErr)
		params := actionURL.Query()
		assert.Equal(t, coreScopeList+" "+jsmOpsScopeList, params.Get("scope"))
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

		// A comfortably-valid token isn't refreshed, but the next check is still scheduled.
		require.Len(t, integrationContext.ResyncRequests, 1)
		assert.Len(t, httpContext.Requests, 2)
	})

	// Regression test: Atlassian has no incremental-consent mechanism, so a token granted
	// without JSM Ops scopes never gains them on its own - turning enableOpsFeatures on for an
	// already-connected integration must prompt a fresh authorize round trip instead of silently
	// doing nothing, which previously left incident/alert/heartbeat actions failing forever.
	t.Run("ops features enabled after connecting - stays ready but prompts reconnect", func(t *testing.T) {
		integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{OpsScopesRequested: false})
		integrationContext.Configuration = map[string]any{
			"clientId":          "client-1",
			"clientSecret":      "secret-1",
			"enableOpsFeatures": true,
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`))},
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			HTTP:          httpContext,
			Integration:   integrationContext,
			Logger:        newLogger(),
		})

		require.NoError(t, err)
		// The connection itself keeps working with its existing (narrower) scope in the meantime.
		assert.Equal(t, "ready", integrationContext.State)

		require.NotNil(t, integrationContext.BrowserAction)
		actionURL, parseErr := url.Parse(integrationContext.BrowserAction.URL)
		require.NoError(t, parseErr)
		assert.Equal(t, coreScopeList+" "+jsmOpsScopeList, actionURL.Query().Get("scope"))

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		// Pending records what the authorize URL asked for; Requested stays false until callback
		// succeeds — otherwise the next Sync would clear the prompt while the token still lacks
		// ops scopes.
		assert.True(t, metadata.OpsScopesPending)
		assert.False(t, metadata.OpsScopesRequested)
	})

	// Regression test: requestAuthorization used to set OpsScopesRequested when building the
	// authorize URL. The next Sync always RemoveBrowserAction'd first, then skipped re-prompt
	// because the flag was already true — permanently clearing the ops reconnect while the
	// token still lacked those scopes.
	t.Run("ops reconnect prompt survives a subsequent Sync until callback succeeds", func(t *testing.T) {
		integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{OpsScopesRequested: false})
		integrationContext.Configuration = map[string]any{
			"clientId":          "client-1",
			"clientSecret":      "secret-1",
			"enableOpsFeatures": true,
		}

		syncOnce := func() {
			httpContext := &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`))},
				},
			}
			err := integration.Sync(core.SyncContext{
				BaseURL:       "https://sp.example.com",
				Configuration: integrationContext.Configuration,
				HTTP:          httpContext,
				Integration:   integrationContext,
				Logger:        newLogger(),
			})
			require.NoError(t, err)
		}

		syncOnce()
		require.NotNil(t, integrationContext.BrowserAction)
		firstURL := integrationContext.BrowserAction.URL

		syncOnce()
		require.NotNil(t, integrationContext.BrowserAction, "ops reconnect prompt must come back after Sync clears browser actions")
		actionURL, parseErr := url.Parse(integrationContext.BrowserAction.URL)
		require.NoError(t, parseErr)
		assert.Equal(t, coreScopeList+" "+jsmOpsScopeList, actionURL.Query().Get("scope"))
		assert.NotEmpty(t, firstURL)

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		assert.False(t, metadata.OpsScopesRequested)
		assert.True(t, metadata.OpsScopesPending)
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

	// Regression test: a metadata reload failure with an otherwise-valid connection (a stale
	// "authorize" browser action left over from before OAuth completed) must not leave that
	// browser action attached - the OAuth connection itself is fine, so there's nothing left to
	// authorize, only a metadata retry on the next sync.
	t.Run("metadata failure with valid credentials clears a stale browser action", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		integrationContext.BrowserAction = &core.BrowserAction{
			Description: "Click Continue to authorize SuperPlane to access your Jira site.",
			URL:         "https://auth.atlassian.com/authorize?...",
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"server error"}`))},
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
		assert.Contains(t, integrationContext.StateDescription, "listing projects")
		assert.Nil(t, integrationContext.BrowserAction)
	})

	t.Run("token close to expiring is proactively refreshed", func(t *testing.T) {
		integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{
			CloudID:              testCloudID,
			AccessTokenExpiresAt: time.Now().Add(time.Minute).Format(time.RFC3339),
		})
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
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

		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Equal(t, "new-access", accessToken)
		assert.Equal(t, TokenURL, httpContext.Requests[0].URL.String())
		require.Len(t, integrationContext.ResyncRequests, 1)
	})

	t.Run("refresh failure with a still-valid token retries later instead of erroring", func(t *testing.T) {
		integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{
			CloudID:              testCloudID,
			AccessTokenExpiresAt: time.Now().Add(time.Minute).Format(time.RFC3339),
		})
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			HTTP:        httpContext,
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		// A concurrent Sync racing to refresh the same rotating token can make this exchange fail
		// even though the credentials are fine; the still-valid access token survives the retry.
		require.NoError(t, err)
		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Equal(t, testAccessToken, accessToken)
		require.Len(t, integrationContext.ResyncRequests, 1)
		assert.Less(t, integrationContext.ResyncRequests[0], time.Minute+time.Second)
	})

	t.Run("refresh failure with an expired token clears secrets and errors", func(t *testing.T) {
		integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{
			CloudID:              testCloudID,
			AccessTokenExpiresAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
		})
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:     "https://sp.example.com",
			HTTP:        httpContext,
			Integration: integrationContext,
			Logger:      newLogger(),
		})

		require.ErrorContains(t, err, "failed to refresh token")
		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		refreshToken, _ := findSecret(integrationContext, SecretOAuthRefreshToken)
		assert.Empty(t, accessToken)
		assert.Empty(t, refreshToken)
	})

	// Regression test: near/past expiry is exactly when a concurrent Sync or 401-driven refresh
	// (Client.recoverFromUnauthorized) is most likely to win the same race, since both trigger in
	// the same window. This proactive refresh failing (its own refresh token was single-use and
	// already spent by the winner) must adopt what the winner stored, not wipe valid credentials.
	t.Run("refresh failure near expiry adopts a concurrently-rotated token instead of wiping it", func(t *testing.T) {
		integrationContext := newAuthorizedIntegrationWithMetadata(Metadata{
			CloudID:              testCloudID,
			AccessTokenExpiresAt: time.Now().Add(-time.Minute).Format(time.RFC3339),
		})
		integrationContext.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}

		httpContext := &raceSimulatingHTTPContext{
			integration: integrationContext,
			responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
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
		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Equal(t, "winner-access", accessToken)
		require.Len(t, integrationContext.ResyncRequests, 1)
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
		assert.False(t, metadata.OpsScopesRequested)
		assert.False(t, metadata.OpsScopesPending)
	})

	// Regression test: OpsScopesRequested must be committed from OpsScopesPending only after a
	// successful callback — that is what stops the ops reconnect prompt, not building the
	// authorize URL itself.
	t.Run("callback with pending ops scopes commits OpsScopesRequested", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
			Metadata: Metadata{State: &state, OpsScopesPending: true},
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

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		assert.True(t, metadata.OpsScopesRequested)
		assert.False(t, metadata.OpsScopesPending)
	})

	// Regression test: a token response missing a refresh token means this connection could
	// never refresh itself once the access token expires - fail the connect immediately with an
	// actionable error instead of reporting success and leaving a time bomb for an hour later.
	t.Run("callback response without a refresh token errors instead of connecting", func(t *testing.T) {
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
					`{"access_token":"cb-access","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net"}]`,
				))},
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
		assert.Contains(t, integrationContext.StateDescription, "no refresh token")

		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Empty(t, accessToken, "must not store the access token without a refresh token")
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

		// Tokens are not stored on this failure, so the next Sync shows the authorize
		// button again instead of getting stuck with an access token but no cloud id.
		accessToken, _ := findSecret(integrationContext, SecretOAuthAccessToken)
		assert.Empty(t, accessToken)
	})

	// Regression test: the OAuth exchange and site resolution already succeeded by this point,
	// so a metadata reload failure must not leave the "authorize" browser action attached - only
	// a metadata retry is needed, not re-authorization.
	t.Run("metadata failure after a successful exchange clears the browser action", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-1",
				"clientSecret": "secret-1",
			},
			Metadata: Metadata{State: &state},
			BrowserAction: &core.BrowserAction{
				Description: "Click Continue to authorize SuperPlane to access your Jira site.",
				URL:         "https://auth.atlassian.com/authorize?...",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"cb-access","refresh_token":"cb-refresh","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net"}]`,
				))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"server error"}`))},
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
		assert.Contains(t, integrationContext.StateDescription, "failed to load workspace data")
		assert.Nil(t, integrationContext.BrowserAction)
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

func Test__Jira__HandleHook(t *testing.T) {
	integration := &Jira{}

	t.Run("refreshes the shared webhook and reschedules itself", func(t *testing.T) {
		webhookID := int64(1000)
		integrationCtx := newAuthorizedIntegrationWithMetadata(Metadata{WebhookID: &webhookID})
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		err := integration.HandleHook(core.IntegrationHookContext{
			Name:        refreshWebhookHookName,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Logger:      newLogger(),
		})
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/rest/api/3/webhook/refresh")
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), "1000")

		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, refreshWebhookHookName, integrationCtx.ActionRequests[0].ActionName)
		assert.Equal(t, webhookRefreshInterval, integrationCtx.ActionRequests[0].Interval)
	})

	t.Run("no-op and no reschedule when the webhook has since been removed", func(t *testing.T) {
		integrationCtx := newAuthorizedIntegration()
		httpCtx := &contexts.HTTPContext{}

		err := integration.HandleHook(core.IntegrationHookContext{
			Name:        refreshWebhookHookName,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Logger:      newLogger(),
		})
		require.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
		assert.Empty(t, integrationCtx.ActionRequests)
	})

	// Regression test: a transient refresh failure must still reschedule itself (with a much
	// shorter retry interval), rather than permanently ending the loop and letting the webhook
	// expire 30 days later with nothing left to retry it.
	t.Run("refresh failure is surfaced but still retries sooner", func(t *testing.T) {
		webhookID := int64(1000)
		integrationCtx := newAuthorizedIntegrationWithMetadata(Metadata{WebhookID: &webhookID})
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["not found"]}`))},
			},
		}

		err := integration.HandleHook(core.IntegrationHookContext{
			Name:        refreshWebhookHookName,
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Logger:      newLogger(),
		})
		require.ErrorContains(t, err, "failed to refresh Jira webhook")

		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, refreshWebhookHookName, integrationCtx.ActionRequests[0].ActionName)
		assert.Equal(t, webhookRefreshRetryInterval, integrationCtx.ActionRequests[0].Interval)
		assert.Less(t, webhookRefreshRetryInterval, webhookRefreshInterval)
	})

	t.Run("unknown hook name errors", func(t *testing.T) {
		err := integration.HandleHook(core.IntegrationHookContext{
			Name:        "bogus",
			Integration: newAuthorizedIntegration(),
			Logger:      newLogger(),
		})
		require.ErrorContains(t, err, "unknown hook")
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
	require.Len(t, triggers, 2)
	assert.Equal(t, "jira.onIssue", triggers[0].Name())
	assert.Equal(t, "jira.onIssueComment", triggers[1].Name())
}
