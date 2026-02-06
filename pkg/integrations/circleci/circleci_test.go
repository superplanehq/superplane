package circleci

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

func Test__CircleCI__Sync(t *testing.T) {
	c := &CircleCI{}

	t.Run("success verifying API token -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"user-123","login":"testuser","name":"Test User"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"apiToken": "token-123",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://circleci.com/api/v2/me", httpContext.Requests[0].URL.String())
		assert.Equal(t, "token-123", httpContext.Requests[0].Header.Get("Circle-Token"))
	})

	t.Run("failure verifying API token -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"apiToken": "invalid-token",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
	})
}

func Test__CircleCI__CompareWebhookConfig(t *testing.T) {
	c := &CircleCI{}

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
				ProjectSlug: "gh/username/repo",
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different project slugs",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo1",
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo2",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"projectSlug": "gh/username/repo",
			},
			configB: map[string]any{
				"projectSlug": "gh/username/repo",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := c.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}
