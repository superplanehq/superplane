package semaphore

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SemaphoreWebhookHandler__CompareConfig(t *testing.T) {
	handler := &SemaphoreWebhookHandler{}

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
				Project: "my-project",
			},
			configB: common.WebhookConfiguration{
				Project: "my-project",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different projects",
			configA: common.WebhookConfiguration{
				Project: "my-project",
			},
			configB: common.WebhookConfiguration{
				Project: "other-project",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"project": "my-project",
			},
			configB: map[string]any{
				"project": "my-project",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: common.WebhookConfiguration{
				Project: "my-project",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: common.WebhookConfiguration{
				Project: "my-project",
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

func Test__SemaphoreWebhookHandler__Cleanup(t *testing.T) {
	t.Run("ignores missing notification and deletes secret", func(t *testing.T) {
		handler := &SemaphoreWebhookHandler{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				semaphoreResponse(http.StatusNotFound, `{"code":5,"message":"Notification not found"}`),
				semaphoreResponse(http.StatusNoContent, ""),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: semaphoreIntegrationContext(),
			Webhook: &contexts.WebhookContext{
				Metadata: WebhookMetadata{
					Secret:       WebhookSecretMetadata{Name: "superplane-webhook-secret"},
					Notification: WebhookNotificationMetadata{ID: "notification-id"},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Equal(t, "/api/v1alpha/notifications/notification-id", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[1].Method)
		assert.Equal(t, "/api/v1beta/secrets/superplane-webhook-secret", httpCtx.Requests[1].URL.Path)
	})

	t.Run("ignores missing secret", func(t *testing.T) {
		handler := &SemaphoreWebhookHandler{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				semaphoreResponse(http.StatusNoContent, ""),
				semaphoreResponse(http.StatusNotFound, `{"code":5,"message":"Secret not found"}`),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpCtx,
			Integration: semaphoreIntegrationContext(),
			Webhook: &contexts.WebhookContext{
				Metadata: WebhookMetadata{
					Secret:       WebhookSecretMetadata{Name: "superplane-webhook-secret"},
					Notification: WebhookNotificationMetadata{ID: "notification-id"},
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
	})
}

func semaphoreIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		NewSetupFlow: true,
		CurrentProperties: map[string]any{
			"organizationUrl": "https://example.semaphoreci.com",
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			"apiToken": {Name: "apiToken", Value: []byte("test-token")},
		},
	}
}

func semaphoreResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
