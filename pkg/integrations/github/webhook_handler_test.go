package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__GitHubWebhookHandler__Setup(t *testing.T) {
	t.Run("registers a repository hook", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusCreated, `{"id":456,"name":"web"}`),
			},
		}

		metadata, err := (&GitHubWebhookHandler{}).Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://app.example/api/v1/webhooks/webhook-id",
				Secret:        []byte("local-webhook-secret"),
				Configuration: common.WebhookConfiguration{EventType: "pull_request", Repository: "hello"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, &Webhook{ID: 456, WebhookName: "web"}, metadata)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello/hooks", httpCtx.Requests[0].URL.Path)
	})

	for _, testCase := range []struct {
		name          string
		responses     []*http.Response
		expectedID    int64
		expectedError string
	}{
		{
			name:       "updates the saved repository hook",
			responses:  []*http.Response{mocks.GitHubResponse(http.StatusOK, `{"id":456,"name":"web"}`)},
			expectedID: 456,
		},
		{
			name:          "returns repository hook permission errors without creating a hook",
			responses:     []*http.Response{mocks.GitHubResponse(http.StatusForbidden, `{"message":"Forbidden"}`)},
			expectedError: "error updating webhook",
		},
		{
			name:          "returns repository hook server errors without creating a hook",
			responses:     []*http.Response{mocks.GitHubResponse(http.StatusInternalServerError, `{"message":"Internal Server Error"}`)},
			expectedError: "error updating webhook",
		},
		{
			name: "recreates a deleted repository hook",
			responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
				mocks.GitHubResponse(http.StatusCreated, `{"id":789,"name":"web"}`),
			},
			expectedID: 789,
		},
		{
			name: "returns replacement hook creation errors",
			responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
				mocks.GitHubResponse(http.StatusForbidden, `{"message":"Forbidden"}`),
			},
			expectedError: "error creating webhook",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: testCase.responses,
			}
			metadata, err := (&GitHubWebhookHandler{}).Setup(core.WebhookHandlerContext{
				HTTP:        httpCtx,
				Integration: mocks.IntegrationContextForNewSetupFlow(),
				Webhook: &contexts.WebhookContext{
					URL:      "https://app.example/api/v1/webhooks/webhook-id",
					Secret:   []byte("local-webhook-secret"),
					Metadata: map[string]any{"id": float64(456), "name": "web"},
					Configuration: common.WebhookConfiguration{
						Repository: "testhq/hello",
						EventTypes: []string{"pull_request", "pull_request_review"},
					},
				},
			})

			require.Len(t, httpCtx.Requests, len(testCase.responses))
			assert.Equal(t, http.MethodPatch, httpCtx.Requests[0].Method)
			assert.Equal(t, "/repos/testhq/hello/hooks/456", httpCtx.Requests[0].URL.Path)
			if len(httpCtx.Requests) == 2 {
				assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
				assert.Equal(t, "/repos/testhq/hello/hooks", httpCtx.Requests[1].URL.Path)
			}
			for _, request := range httpCtx.Requests {
				var payload struct {
					Events []string          `json:"events"`
					Config map[string]string `json:"config"`
				}
				require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
				assert.ElementsMatch(t, []string{"pull_request", "pull_request_review"}, payload.Events)
				assert.Equal(t, "https://app.example/api/v1/webhooks/webhook-id", payload.Config["url"])
				assert.Equal(t, "local-webhook-secret", payload.Config["secret"])
			}
			if testCase.expectedError != "" {
				require.ErrorContains(t, err, testCase.expectedError)
				assert.Nil(t, metadata)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, &Webhook{ID: testCase.expectedID, WebhookName: "web"}, metadata)
		})
	}

	t.Run("registers one hook per workspace repository", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusCreated, `{"id":456,"name":"web"}`),
				mocks.GitHubResponse(http.StatusCreated, `{"id":457,"name":"web"}`),
			},
		}

		metadata, err := (&GitHubWebhookHandler{}).Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			Webhook: &contexts.WebhookContext{
				URL:    "https://app.example/api/v1/webhooks/webhook-id",
				Secret: []byte("local-webhook-secret"),
				Configuration: common.WebhookConfiguration{
					EventType:    "pull_request",
					Repository:   "testhq/hello",
					Repositories: []string{"testhq/hello", "testhq/world"},
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, &Webhook{
			ID:          456,
			WebhookName: "web",
			Hooks: []RepositoryWebhook{
				{Repository: "testhq/hello", ID: 456, Name: "web"},
				{Repository: "testhq/world", ID: 457, Name: "web"},
			},
		}, metadata)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "/repos/testhq/hello/hooks", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "/repos/testhq/world/hooks", httpCtx.Requests[1].URL.Path)
	})

	t.Run("removes created hooks when a later repository fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusCreated, `{"id":456,"name":"web"}`),
				mocks.GitHubResponse(http.StatusForbidden, `{"message":"Forbidden"}`),
				mocks.GitHubResponse(http.StatusNoContent, ""),
			},
		}

		_, err := (&GitHubWebhookHandler{}).Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			Webhook: &contexts.WebhookContext{
				URL:    "https://app.example/api/v1/webhooks/webhook-id",
				Secret: []byte("local-webhook-secret"),
				Configuration: common.WebhookConfiguration{
					EventType:    "pull_request",
					Repositories: []string{"testhq/hello", "testhq/world"},
				},
			},
		})

		require.Error(t, err)
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[2].Method)
		assert.Equal(t, "/repos/testhq/hello/hooks/456", httpCtx.Requests[2].URL.Path)
	})

	// Hosted GitHub App metadata must not skip hook registration. The App
	// webhook endpoint no longer delivers repository events to nodes.
	t.Run("does not skip hook registration for hosted app metadata", func(t *testing.T) {
		t.Setenv(config.EnvGitHubAppID, "12345")
		t.Setenv(config.EnvGitHubAppSlug, "superplane")
		t.Setenv(config.EnvGitHubAppPrivateKey, "private-key")
		t.Setenv(config.EnvGitHubAppWebhookSecret, "app-webhook-secret")

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusCreated, `{"id":789,"name":"web"}`),
			},
		}
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		integrationCtx.Metadata = map[string]any{"hostedApp": true}

		metadata, err := (&GitHubWebhookHandler{}).Setup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Webhook: &contexts.WebhookContext{
				URL:           "https://app.example/api/v1/webhooks/webhook-id",
				Secret:        []byte("local-webhook-secret"),
				Configuration: common.WebhookConfiguration{EventType: "pull_request", Repository: "hello"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, &Webhook{ID: 789, WebhookName: "web"}, metadata)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/testhq/hello/hooks", httpCtx.Requests[0].URL.Path)
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
			name: "same repository set in a different order and case",
			configA: common.WebhookConfiguration{
				EventType:    "pull_request",
				Repositories: []string{"acme/api", "acme/web"},
			},
			configB: common.WebhookConfiguration{
				EventType:    "pull_request",
				Repositories: []string{"Acme/Web", "acme/api"},
			},
			expectEqual: true,
		},
		{
			name: "repository set gains a repository",
			configA: common.WebhookConfiguration{
				EventType:    "pull_request",
				Repositories: []string{"acme/api"},
			},
			configB: common.WebhookConfiguration{
				EventType:    "pull_request",
				Repositories: []string{"acme/api", "acme/web"},
			},
			expectEqual: false,
		},
		{
			name: "single repository differs from a repository set",
			configA: common.WebhookConfiguration{
				EventType:  "pull_request",
				Repository: "acme/api",
			},
			configB: common.WebhookConfiguration{
				EventType:    "pull_request",
				Repository:   "acme/api",
				Repositories: []string{"acme/api", "acme/web"},
			},
			expectEqual: false,
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

	t.Run("removes the hook in every workspace repository", func(t *testing.T) {
		handler := &GitHubWebhookHandler{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNoContent, ""),
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			Webhook: &contexts.WebhookContext{
				Metadata: map[string]any{
					"id":   123,
					"name": "web",
					"hooks": []any{
						map[string]any{"repository": "testhq/hello", "id": 123, "name": "web"},
						map[string]any{"repository": "testhq/world", "id": 124, "name": "web"},
					},
				},
				Configuration: common.WebhookConfiguration{
					Repository:   "testhq/hello",
					Repositories: []string{"testhq/hello", "testhq/world"},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "/repos/testhq/hello/hooks/123", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "/repos/testhq/world/hooks/124", httpCtx.Requests[1].URL.Path)
	})

	t.Run("does not skip hook removal for hosted app metadata", func(t *testing.T) {
		handler := &GitHubWebhookHandler{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNoContent, ""),
			},
		}
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		integrationCtx.Metadata = map[string]any{"hostedApp": true}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
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
