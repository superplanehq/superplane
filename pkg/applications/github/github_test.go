package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GitHub__Setup(t *testing.T) {
	g := &GitHub{}

	t.Run("personal scope", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		require.NoError(t, g.Sync(core.SyncContext{AppInstallation: appCtx}))

		//
		// Browser action is created
		//
		require.NotNil(t, appCtx.BrowserAction)
		assert.Equal(t, appCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, appCtx.BrowserAction.Description)
		assert.Equal(t, appCtx.BrowserAction.URL, "https://github.com/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, appCtx.Metadata)
		metadata := appCtx.Metadata.(Metadata)
		assert.Empty(t, metadata.Owner)
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("organization scope", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			Configuration:   Configuration{Organization: "testhq"},
			AppInstallation: appCtx,
		}))

		//
		// Browser action is created
		//
		require.NotNil(t, appCtx.BrowserAction)
		assert.Equal(t, appCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, appCtx.BrowserAction.Description)
		assert.Equal(t, appCtx.BrowserAction.URL, "https://github.com/organizations/testhq/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, appCtx.Metadata)
		metadata := appCtx.Metadata.(Metadata)
		assert.Equal(t, metadata.Owner, "testhq")
		assert.NotEmpty(t, metadata.State)
	})
}

func Test__GitHub__CompareWebhookConfig(t *testing.T) {
	g := &GitHub{}

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
				EventType:  "push",
				Repository: "superplane",
			},
			configB: WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different event types",
			configA: WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: WebhookConfiguration{
				EventType:  "pull_request",
				Repository: "superplane",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different repositories",
			configA: WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: WebhookConfiguration{
				EventType:  "push",
				Repository: "other-repo",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "both fields different",
			configA: WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			configB: WebhookConfiguration{
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
			configB: WebhookConfiguration{
				EventType:  "push",
				Repository: "superplane",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
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
			equal, err := g.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}
