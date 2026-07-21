package linear

import (
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

func Test__Linear__Sync(t *testing.T) {
	integration := &Linear{}

	t.Run("no client credentials - setup wizard with pre-filled app form", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{Configuration: map[string]any{}}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Contains(t, integrationContext.BrowserAction.Description, "OAuth application form")

		actionURL, parseErr := url.Parse(integrationContext.BrowserAction.URL)
		require.NoError(t, parseErr)
		assert.Equal(t, "linear.app", actionURL.Host)
		assert.Equal(t, "/settings/api/applications/new", actionURL.Path)

		params := actionURL.Query()
		assert.Equal(t, "SuperPlane", params.Get("oauth.client_name"))
		assert.Equal(t, "SuperPlane", params.Get("developer.name"))
		assert.Equal(t, "https://sp.example.com", params.Get("oauth.client_uri"))
		assert.Contains(t, params.Get("oauth.redirect_uris"), "/api/v1/integrations/")
		assert.Contains(t, params.Get("oauth.redirect_uris"), "/callback")
	})

	t.Run("missing client secret - setup wizard", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{"clientId": testClientID},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Contains(t, integrationContext.BrowserAction.URL, "/settings/api/applications/new")
	})

	t.Run("credentials but no access token - authorize button", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     testClientID,
				"clientSecret": testClientSecret,
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Contains(t, integrationContext.BrowserAction.Description, "authorize SuperPlane")

		actionURL, parseErr := url.Parse(integrationContext.BrowserAction.URL)
		require.NoError(t, parseErr)
		assert.Equal(t, "linear.app", actionURL.Host)
		assert.Equal(t, "/oauth/authorize", actionURL.Path)

		params := actionURL.Query()
		assert.Equal(t, testClientID, params.Get("client_id"))
		assert.Equal(t, "code", params.Get("response_type"))
		assert.Equal(t, "read,write,admin", params.Get("scope"))
		assert.Equal(t, "user", params.Get("actor"))
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
				"clientId":     testClientID,
				"clientSecret": testClientSecret,
			},
			Metadata: Metadata{State: &state},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.NotNil(t, integrationContext.BrowserAction)
		assert.Contains(t, integrationContext.BrowserAction.URL, "state=existing-state")
	})

	t.Run("access token present - refreshes and becomes ready", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"access_token":"new-access","refresh_token":"new-refresh","token_type":"Bearer","expires_in":86399}`),
				jsonResponse(`{"data":{"viewer":{"id":"u1","name":"Jane Doe","displayName":"jane","email":"jane@example.com"},"organization":{"id":"o1","name":"Acme","urlKey":"acme"}}}`),
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ENG","name":"Engineering"}],"pageInfo":{"hasNextPage":false}}}}`),
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			HTTP:          httpContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationContext.State)
		assert.Nil(t, integrationContext.BrowserAction)

		// Rotated token pair replaces the stored secrets.
		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		refreshToken, _ := findSecret(integrationContext, OAuthRefreshToken)
		assert.Equal(t, "new-access", accessToken)
		assert.Equal(t, "new-refresh", refreshToken)

		// Resync is scheduled at half the 24h token lifetime.
		require.Len(t, integrationContext.ResyncRequests, 1)

		metadata, ok := integrationContext.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.User)
		assert.Equal(t, "Jane Doe", metadata.User.Name)
		assert.Equal(t, "Acme", metadata.Organization)
		assert.Equal(t, "acme", metadata.URLKey)
		require.Len(t, metadata.Teams, 1)
		assert.Equal(t, "ENG", metadata.Teams[0].Key)

		// The refresh request went to the token endpoint, form-encoded.
		require.GreaterOrEqual(t, len(httpContext.Requests), 1)
		tokenRequest := httpContext.Requests[0]
		assert.Equal(t, TokenURL, tokenRequest.URL.String())
		assert.Equal(t, "application/x-www-form-urlencoded", tokenRequest.Header.Get("Content-Type"))
	})

	t.Run("access token without refresh token routes back to authorization", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()
		delete(integrationContext.CurrentSecrets, OAuthRefreshToken)

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			HTTP:          &contexts.HTTPContext{},
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.ErrorContains(t, err, "no refresh token")

		// The dead-end access token is cleared, so the next
		// sync shows the authorize button again.
		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		assert.Empty(t, accessToken)
	})

	t.Run("credential verification failure marks the integration errored", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"access_token":"new-access","refresh_token":"new-refresh","token_type":"Bearer","expires_in":86399}`),
				jsonResponse(`{"errors":[{"message":"Authentication required, not authenticated"}]}`),
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			HTTP:          httpContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		// Sync reports the failure through the integration state, keeping the
		// refreshed tokens for the next attempt.
		require.NoError(t, err)
		assert.Equal(t, "error", integrationContext.State)
		assert.Contains(t, integrationContext.StateDescription, "error verifying Linear credentials")

		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		assert.Equal(t, "new-access", accessToken)
	})

	t.Run("refresh failure clears tokens and errors", func(t *testing.T) {
		integrationContext := newAuthorizedIntegration()

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: http.NoBody},
			},
		}

		err := integration.Sync(core.SyncContext{
			BaseURL:       "https://sp.example.com",
			Configuration: integrationContext.Configuration,
			Integration:   integrationContext,
			HTTP:          httpContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.ErrorContains(t, err, "failed to refresh token")

		// Cleared secrets route the user back to the authorize step next sync.
		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		refreshToken, _ := findSecret(integrationContext, OAuthRefreshToken)
		assert.Empty(t, accessToken)
		assert.Empty(t, refreshToken)
	})
}

func Test__Linear__HandleRequest(t *testing.T) {
	integration := &Linear{}

	t.Run("non-callback path -> 404", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Request:     httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/other", nil),
			Response:    recorder,
			Integration: newAuthorizedIntegration(),
			Logger:      logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("valid callback exchanges code, stores tokens and becomes ready", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     testClientID,
				"clientSecret": testClientSecret,
			},
			Metadata: Metadata{State: &state},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"access_token":"cb-access","refresh_token":"cb-refresh","token_type":"Bearer","expires_in":86399}`),
				jsonResponse(`{"data":{"viewer":{"id":"u1","name":"Jane Doe"},"organization":{"id":"o1","name":"Acme","urlKey":"acme"}}}`),
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ENG","name":"Engineering"}],"pageInfo":{"hasNextPage":false}}}}`),
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
			Logger:         logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "ready", integrationContext.State)

		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		refreshToken, _ := findSecret(integrationContext, OAuthRefreshToken)
		assert.Equal(t, "cb-access", accessToken)
		assert.Equal(t, "cb-refresh", refreshToken)

		require.Len(t, integrationContext.ResyncRequests, 1)

		// Token exchange used the authorization code grant with the callback redirect URI.
		require.GreaterOrEqual(t, len(httpContext.Requests), 1)
		body := readAndRestoreBody(t, httpContext.Requests[0])
		form, parseErr := url.ParseQuery(string(body))
		require.NoError(t, parseErr)
		assert.Equal(t, "authorization_code", form.Get("grant_type"))
		assert.Equal(t, "auth-code", form.Get("code"))
		assert.True(t, strings.HasSuffix(form.Get("redirect_uri"), "/callback"))
	})

	t.Run("metadata failure after token storage redirects with an error state", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     testClientID,
				"clientSecret": testClientSecret,
			},
			Metadata: Metadata{State: &state},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"access_token":"cb-access","refresh_token":"cb-refresh","token_type":"Bearer","expires_in":86399}`),
				jsonResponse(`{"errors":[{"message":"workspace temporarily unavailable"}]}`),
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
			Logger:         logrus.NewEntry(logrus.New()),
		})

		// The user lands back on the settings page instead of a bare 500,
		// with the failure surfaced through the integration state.
		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "error", integrationContext.State)
		assert.Contains(t, integrationContext.StateDescription, "failed to load workspace data")

		// The token pair survives, so the scheduled resync can recover alone.
		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		refreshToken, _ := findSecret(integrationContext, OAuthRefreshToken)
		assert.Equal(t, "cb-access", accessToken)
		assert.Equal(t, "cb-refresh", refreshToken)
		require.Len(t, integrationContext.ResyncRequests, 1)
	})

	t.Run("state mismatch does not store tokens", func(t *testing.T) {
		state := "expected-state"
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     testClientID,
				"clientSecret": testClientSecret,
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
			Logger:         logrus.NewEntry(logrus.New()),
		})

		// Errors redirect back to the settings page rather than failing hard.
		assert.Equal(t, http.StatusSeeOther, recorder.Code)

		accessToken, _ := findSecret(integrationContext, OAuthAccessToken)
		assert.Empty(t, accessToken)
	})
}

func Test__Linear__Definition(t *testing.T) {
	integration := &Linear{}

	assert.Equal(t, "linear", integration.Name())
	assert.Equal(t, "Linear", integration.Label())
	assert.Equal(t, "linear", integration.Icon())

	actions := integration.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "linear.createIssue", actions[0].Name())

	triggers := integration.Triggers()
	require.Len(t, triggers, 1)
	assert.Equal(t, "linear.onIssue", triggers[0].Name())
}
