package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

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

func Test__GitHub__HandleWebhook(t *testing.T) {
	t.Run("returns not found when new setup flow is missing app webhook secret", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		integration := mocks.IntegrationContextForNewSetupFlow()
		delete(integration.CurrentSecrets, common.SecretAppWebhookSecret)

		(&GitHub{}).handleWebhook(githubWebhookRequestContext(recorder, integration, []byte("{}")))

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.NotContains(t, recorder.Body.String(), common.SecretAppWebhookSecret)
	})

	t.Run("keeps invalid signature status when new setup flow has app webhook secret", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		integration := mocks.IntegrationContextForNewSetupFlow()
		require.NoError(t, integration.SetSecret(common.SecretAppWebhookSecret, []byte("webhook-secret")))

		(&GitHub{}).handleWebhook(githubWebhookRequestContext(recorder, integration, []byte("{}")))

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("keeps legacy missing webhook secret behavior", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		integration := mocks.IntegrationContextForLegacySetupFlow(githubPrivateKeyPEM(t))

		(&GitHub{}).handleWebhook(githubWebhookRequestContext(recorder, integration, []byte("{}")))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("keeps unexpected new setup secret retrieval failures as internal errors", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		integration := &failingSecretIntegrationContext{
			IntegrationContext: mocks.IntegrationContextForNewSetupFlow(),
			err:                errors.New("storage unavailable"),
		}

		(&GitHub{}).handleWebhook(githubWebhookRequestContext(recorder, integration, []byte("{}")))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("keeps malformed payload behavior", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		secret := "webhook-secret"
		body := []byte("not-json")
		integration := mocks.IntegrationContextForNewSetupFlow()
		require.NoError(t, integration.SetSecret(common.SecretAppWebhookSecret, []byte(secret)))

		ctx := githubWebhookRequestContext(recorder, integration, body)
		ctx.Request.Header.Set("X-GitHub-Event", "installation")
		ctx.Request.Header.Set("X-Hub-Signature-256", githubSignature(secret, body))

		(&GitHub{}).handleWebhook(ctx)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
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

func githubWebhookRequestContext(
	recorder *httptest.ResponseRecorder,
	integration core.IntegrationContext,
	body []byte,
) core.HTTPRequestContext {
	return core.HTTPRequestContext{
		Logger:      logrus.NewEntry(logrus.New()),
		Request:     httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/webhook", bytes.NewReader(body)),
		Response:    recorder,
		Integration: integration,
	}
}

func githubSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type failingSecretIntegrationContext struct {
	*contexts.IntegrationContext
	err error
}

func (c *failingSecretIntegrationContext) Secrets() core.IntegrationSecretStorage {
	return failingSecretStorage{err: c.err}
}

type failingSecretStorage struct {
	err error
}

func (s failingSecretStorage) Get(string) (string, error) {
	return "", s.err
}

func (s failingSecretStorage) Delete(string) error {
	return s.err
}

func (s failingSecretStorage) Create(core.IntegrationSecretDefinition) error {
	return s.err
}

func (s failingSecretStorage) CreateMany([]core.IntegrationSecretDefinition) error {
	return s.err
}

func (s failingSecretStorage) Update(string, string) error {
	return s.err
}
