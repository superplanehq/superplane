package semaphore

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Semaphore__Sync(t *testing.T) {
	s := &Semaphore{}

	t.Run("success listing projects -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("[]")),
				},
			},
		}

		appInstallation := &contexts.AppInstallationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration:   appInstallation.Configuration,
			HTTP:            httpContext,
			AppInstallation: appInstallation,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appInstallation.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/projects", httpContext.Requests[0].URL.String())
	})

	t.Run("failure listing projects -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("server error")),
				},
			},
		}

		appInstallation := &contexts.AppInstallationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration:   appInstallation.Configuration,
			HTTP:            httpContext,
			AppInstallation: appInstallation,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appInstallation.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/projects", httpContext.Requests[0].URL.String())
	})
}

func Test__Semaphore__CompareWebhookConfig(t *testing.T) {
	s := &Semaphore{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical configurations",
			configA: WebhookConfiguration{
				Project: "my-project",
			},
			configB: WebhookConfiguration{
				Project: "my-project",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different projects",
			configA: WebhookConfiguration{
				Project: "my-project",
			},
			configB: WebhookConfiguration{
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
			configB: WebhookConfiguration{
				Project: "my-project",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Project: "my-project",
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := s.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}
