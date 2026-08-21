package openrouter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// failingSecretsIntegration simulates a transient secret-store failure.
type failingSecretsIntegration struct {
	*contexts.IntegrationContext
}

func (f *failingSecretsIntegration) GetSecrets() ([]core.IntegrationSecret, error) {
	return nil, fmt.Errorf("secret store unavailable")
}

func syncContext(integration *contexts.IntegrationContext, httpContext *contexts.HTTPContext) core.SyncContext {
	return core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: integration.Configuration,
		BaseURL:       "https://app.superplane.test",
		HTTP:          httpContext,
		Integration:   integration,
	}
}

func Test__Sync(t *testing.T) {
	o := &OpenRouter{}

	t.Run("without a key it sends the user to OpenRouter's consent screen", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{}}

		err := o.Sync(syncContext(integration, &contexts.HTTPContext{}))

		require.NoError(t, err)
		require.NotNil(t, integration.BrowserAction)
		assert.Equal(t, "GET", integration.BrowserAction.Method)
		assert.Contains(t, integration.BrowserAction.URL, AuthorizeURL)
		assert.Contains(t, integration.BrowserAction.URL, "code_challenge_method=S256")

		// The callback carries the state, since OpenRouter's authorize endpoint
		// takes no state parameter of its own.
		assert.Contains(t, integration.BrowserAction.URL, "callback_url=")
		assert.Contains(t, integration.BrowserAction.URL, "state")

		// The verifier is a secret, never metadata: metadata reaches the browser.
		verifier, err := findSecret(integration, SecretCodeVerifier)
		require.NoError(t, err)
		assert.NotEmpty(t, verifier)
		assert.NotContains(t, integration.BrowserAction.URL, verifier)

		assert.NotEqual(t, "ready", integration.State)
	})

	// Sync runs on a schedule. Re-minting would rotate the verifier under a
	// consent screen the user already opened, so the exchange would fail after
	// they approved.
	t.Run("a repeated sync keeps the pending flow intact", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{}}

		require.NoError(t, o.Sync(syncContext(integration, &contexts.HTTPContext{})))
		firstURL := integration.BrowserAction.URL
		firstVerifier, err := findSecret(integration, SecretCodeVerifier)
		require.NoError(t, err)
		firstState := integration.Metadata.(Metadata).State

		require.NoError(t, o.Sync(syncContext(integration, &contexts.HTTPContext{})))
		secondVerifier, err := findSecret(integration, SecretCodeVerifier)
		require.NoError(t, err)

		assert.Equal(t, firstVerifier, secondVerifier, "the verifier must survive a background sync")
		assert.Equal(t, firstState, integration.Metadata.(Metadata).State)
		assert.Equal(t, firstURL, integration.BrowserAction.URL)
	})

	t.Run("a fresh connection mints a new flow", func(t *testing.T) {
		first := &contexts.IntegrationContext{Configuration: map[string]any{}}
		second := &contexts.IntegrationContext{Configuration: map[string]any{}}

		require.NoError(t, o.Sync(syncContext(first, &contexts.HTTPContext{})))
		require.NoError(t, o.Sync(syncContext(second, &contexts.HTTPContext{})))

		firstVerifier, _ := findSecret(first, SecretCodeVerifier)
		secondVerifier, _ := findSecret(second, SecretCodeVerifier)
		assert.NotEqual(t, firstVerifier, secondVerifier)
	})

	// Treating a read failure as "not connected" would restart OAuth and
	// disconnect a working integration over a transient error.
	t.Run("a secret read failure does not restart the flow", func(t *testing.T) {
		integration := &failingSecretsIntegration{
			IntegrationContext: &contexts.IntegrationContext{Configuration: map[string]any{}},
		}

		err := o.Sync(core.SyncContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{},
			BaseURL:       "https://app.superplane.test",
			HTTP:          &contexts.HTTPContext{},
			Integration:   integration,
		})

		require.ErrorContains(t, err, "failed to read integration secrets")
		assert.Nil(t, integration.BrowserAction)
	})

	t.Run("with a key it verifies and becomes ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":{"label":"test","usage":0,"is_free_tier":true}}`),
			},
		}
		integration := connectedIntegration(map[string]any{})
		integration.BrowserAction = &core.BrowserAction{URL: "stale"}

		err := o.Sync(syncContext(integration, httpContext))

		require.NoError(t, err)
		assert.Equal(t, "ready", integration.State)
		assert.Nil(t, integration.BrowserAction)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/key")
		assert.Equal(t, "Bearer sk-or-v1-test", httpContext.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, attributionTitle, httpContext.Requests[0].Header.Get("X-Title"))
	})

	t.Run("a revoked key fails the sync", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusUnauthorized, `{"error":{"message":"User not found.","code":401}}`),
			},
		}
		integration := connectedIntegration(map[string]any{})

		err := o.Sync(syncContext(integration, httpContext))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("a bad provisioning key does not block readiness", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":{"label":"test"}}`),
				response(http.StatusForbidden, `{"error":{"message":"Only management keys can fetch activity for an account","code":403}}`),
			},
		}
		integration := connectedIntegration(map[string]any{"managementKey": "sk-or-provisioning"})

		err := o.Sync(syncContext(integration, httpContext))

		require.NoError(t, err)
		assert.Equal(t, "ready", integration.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/activity")
		assert.Equal(t, "Bearer sk-or-provisioning", httpContext.Requests[1].Header.Get("Authorization"))
	})
}

func callbackRequest(target string) *http.Request {
	return httptest.NewRequest(http.MethodGet, target, nil)
}

func Test__HandleRequest__Callback(t *testing.T) {
	o := &OpenRouter{}

	newIntegration := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration:  map[string]any{},
			Metadata:       map[string]any{"state": "expected-state"},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretCodeVerifier: {Name: SecretCodeVerifier, Value: []byte("verifier-123")}},
			BrowserAction:  &core.BrowserAction{URL: "authorize"},
		}
	}

	requestContext := func(integration *contexts.IntegrationContext, httpContext *contexts.HTTPContext, target string, recorder *httptest.ResponseRecorder) core.HTTPRequestContext {
		return core.HTTPRequestContext{
			Logger:         logrus.NewEntry(logrus.New()),
			Request:        callbackRequest(target),
			Response:       recorder,
			BaseURL:        "https://app.superplane.test",
			OrganizationID: "org-1",
			HTTP:           httpContext,
			Integration:    integration,
		}
	}

	t.Run("exchanges the code and stores the key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{response(http.StatusOK, `{"key":"sk-or-v1-issued"}`)},
		}
		integration := newIntegration()
		recorder := httptest.NewRecorder()

		o.HandleRequest(requestContext(integration, httpContext, "/api/v1/integrations/abc/callback?code=the-code&state=expected-state", recorder))

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/auth/keys")

		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"code":"the-code"`)
		assert.Contains(t, string(body), `"code_verifier":"verifier-123"`)
		assert.Contains(t, string(body), `"code_challenge_method":"S256"`)

		key, err := findSecret(integration, SecretAPIKey)
		require.NoError(t, err)
		assert.Equal(t, "sk-or-v1-issued", key)

		// The verifier is single-use.
		verifier, err := findSecret(integration, SecretCodeVerifier)
		require.NoError(t, err)
		assert.Empty(t, verifier)

		assert.Equal(t, "ready", integration.State)
		assert.Nil(t, integration.BrowserAction)
		assert.Equal(t, http.StatusSeeOther, recorder.Code)
	})

	t.Run("rejects a mismatched state without exchanging", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integration := newIntegration()
		recorder := httptest.NewRecorder()

		o.HandleRequest(requestContext(integration, httpContext, "/api/v1/integrations/abc/callback?code=the-code&state=forged", recorder))

		assert.Empty(t, httpContext.Requests)
		key, _ := findSecret(integration, SecretAPIKey)
		assert.Empty(t, key)
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("does not exchange without a code", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integration := newIntegration()
		recorder := httptest.NewRecorder()

		o.HandleRequest(requestContext(integration, httpContext, "/api/v1/integrations/abc/callback?state=expected-state", recorder))

		assert.Empty(t, httpContext.Requests)
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("surfaces an exchange failure without storing a key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{response(http.StatusBadRequest, `{"error":{"message":"Invalid code","code":400}}`)},
		}
		integration := newIntegration()
		recorder := httptest.NewRecorder()

		o.HandleRequest(requestContext(integration, httpContext, "/api/v1/integrations/abc/callback?code=bad&state=expected-state", recorder))

		key, _ := findSecret(integration, SecretAPIKey)
		assert.Empty(t, key)
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("ignores unknown paths", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		o.HandleRequest(requestContext(newIntegration(), &contexts.HTTPContext{}, "/api/v1/integrations/abc/webhook", recorder))

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})
}

func Test__PKCE(t *testing.T) {
	t.Run("the challenge is the unpadded base64url SHA-256 of the verifier", func(t *testing.T) {
		// Test vector from RFC 7636 appendix B.
		assert.Equal(t,
			"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			codeChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
		)
	})

	t.Run("verifiers are unpadded and unique", func(t *testing.T) {
		first, err := newCodeVerifier()
		require.NoError(t, err)
		second, err := newCodeVerifier()
		require.NoError(t, err)

		assert.NotEqual(t, first, second)
		assert.NotContains(t, first, "=")
		assert.GreaterOrEqual(t, len(first), 43)
	})
}

func Test__ListResources__Models(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			response(http.StatusOK, `{"data":[
				{"id":"openai/gpt-4o-mini","name":"OpenAI: GPT-4o-mini"},
				{"id":"anthropic/claude-sonnet-4.5","name":"Anthropic: Claude Sonnet 4.5"},
				{"id":"","name":"broken"}
			]}`),
		},
	}

	o := &OpenRouter{}
	resources, err := o.ListResources(ResourceTypeModel, core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        httpContext,
		Integration: connectedIntegration(map[string]any{}),
	})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "openai/gpt-4o-mini", resources[0].ID)
	assert.Equal(t, "anthropic/claude-sonnet-4.5", resources[1].ID)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/models")
}

func Test__ListResources__Providers(t *testing.T) {
	t.Run("scoped to the selected model", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":{"id":"openai/gpt-4o-mini","endpoints":[
					{"provider_name":"Azure","tag":"azure"},
					{"provider_name":"OpenAI","tag":"openai"},
					{"provider_name":"Azure","tag":"azure/swedencentral"}
				]}}`),
			},
		}

		o := &OpenRouter{}
		resources, err := o.ListResources(ResourceTypeProvider, core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: connectedIntegration(map[string]any{}),
			Parameters:  map[string]string{"model": "openai/gpt-4o-mini"},
		})

		require.NoError(t, err)

		// The regional Azure endpoint collapses into the azure provider slug,
		// which is the form routing accepts.
		require.Len(t, resources, 2)
		assert.Equal(t, "azure", resources[0].ID)
		assert.Equal(t, "Azure", resources[0].Name)
		assert.Equal(t, "openai", resources[1].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/models/openai/gpt-4o-mini/endpoints")
	})

	t.Run("falls back to every provider without a model", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"data":[
					{"name":"Cerebras","slug":"cerebras"},
					{"name":"Together","slug":"together"}
				]}`),
			},
		}

		o := &OpenRouter{}
		resources, err := o.ListResources(ResourceTypeProvider, core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: connectedIntegration(map[string]any{}),
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "cerebras", resources[0].ID)
		assert.Equal(t, "Cerebras", resources[0].Name)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/providers")
	})
}

func Test__ListResources__UnknownType(t *testing.T) {
	o := &OpenRouter{}
	resources, err := o.ListResources("unknown", core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        &contexts.HTTPContext{},
		Integration: connectedIntegration(map[string]any{}),
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}
