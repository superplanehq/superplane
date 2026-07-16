package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__GitHub__handleWebhook__MissingSecret(t *testing.T) {
	g := &GitHub{}

	newRequestContext := func(integration core.IntegrationContext) (core.HTTPRequestContext, *httptest.ResponseRecorder) {
		recorder := httptest.NewRecorder()
		return core.HTTPRequestContext{
			Logger:      logger.DiscardLogger(),
			Request:     httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/webhook", strings.NewReader("{}")),
			Response:    recorder,
			Integration: integration,
		}, recorder
	}

	//
	// Regression test for issue #5850: an integration whose setup never
	// completed has no webhook secret stored. This must be reported as a 4xx
	// setup problem, not a 500 that pages us via Sentry.
	//
	t.Run("non-legacy setup without appWebhookSecret returns 404", func(t *testing.T) {
		ctx, recorder := newRequestContext(mocks.IntegrationContextForNewSetupFlow())

		g.handleWebhook(ctx)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("legacy setup without webhookSecret returns 404", func(t *testing.T) {
		ctx, recorder := newRequestContext(mocks.IntegrationContextForLegacySetupFlow(githubPrivateKeyPEM(t)))

		g.handleWebhook(ctx)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	//
	// When the secret is present, the handler proceeds to signature validation.
	// A payload with no valid signature is rejected as a bad request (400),
	// confirming the secret-present path is unaffected by the 404 change.
	//
	t.Run("secret present with invalid signature returns 400", func(t *testing.T) {
		integration := mocks.IntegrationContextForNewSetupFlow()
		require.NoError(t, integration.SetSecret(common.SecretAppWebhookSecret, []byte("webhook-secret")))

		ctx, recorder := newRequestContext(integration)

		g.handleWebhook(ctx)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func Test__GitHubWebhookHandler__CompareConfig(t *testing.T) {
	handler := &GitHubWebhookHandler{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical configurations",
			configA: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different event types",
			configA: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: common.WebhookConfiguration{
				EventType:  "pull_request",
				Repository: "superplane",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different repositories",
			configA: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "other-repo",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "both fields different",
			configA: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: common.WebhookConfiguration{
				EventType:  "issues",
				Repository: "other-repo",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"eventType":  "push",
				"repository": "superplane",
			},
			configB: map[string]any{
				"eventType":  "push",
				"repository": "superplane",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: common.WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := handler.CompareConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}

func Test__GitHubWebhookHandler__Cleanup(t *testing.T) {
	t.Run("ignores missing hook", func(t *testing.T) {
		handler := &GitHubWebhookHandler{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			Webhook: &contexts.WebhookContext{
				Metadata:      Webhook{ID: 123},
				Configuration: common.WebhookConfiguration{Repository: "hello"},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello/hooks/123", httpCtx.Requests[0].URL.Path)
	})

	t.Run("ignores missing app installation during token refresh", func(t *testing.T) {
		handler := &GitHubWebhookHandler{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForLegacySetupFlow(githubPrivateKeyPEM(t)),
			Webhook: &contexts.WebhookContext{
				Metadata:      Webhook{ID: 123},
				Configuration: common.WebhookConfiguration{Repository: "hello"},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/app/installations/67890/access_tokens", httpCtx.Requests[0].URL.Path)
	})
}

func githubPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}
