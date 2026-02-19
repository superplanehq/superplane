package rootly

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__RootlyWebhookHandler__CompareConfig(t *testing.T) {
	handler := &RootlyWebhookHandler{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical events",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different events",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.resolved"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "superset of events (A contains all of B)",
			configA: WebhookConfiguration{
				Events: []string{"incident.created", "incident.updated", "incident.resolved"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "subset of events (A does not contain all of B)",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.created", "incident.resolved"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"events": []string{"incident.created", "incident.updated"},
			},
			configB: map[string]any{
				"events": []string{"incident.created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
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
				assert.Error(t, err, "expected error, but got none")
			} else {
				require.NoError(t, err, "did not expect, but got an error")
			}

			assert.Equal(t, tc.expectEqual, equal, "expected config comparison result to match")
		})
	}
}

func Test__RootlyWebhookHandler__Merge(t *testing.T) {
	handler := &RootlyWebhookHandler{}

	current := WebhookConfiguration{
		Events: []string{"incident.created", "incident.updated"},
	}

	requested := WebhookConfiguration{
		Events: []string{"incident_event.created", "incident.updated"},
	}

	merged, changed, err := handler.Merge(current, requested)
	require.NoError(t, err)

	result := WebhookConfiguration{}
	require.NoError(t, mapstructure.Decode(merged, &result))

	assert.ElementsMatch(t, []string{"incident.created", "incident.updated", "incident_event.created"}, result.Events)
	assert.True(t, changed)
}

func Test__SelectWebhookEndpoint(t *testing.T) {
	webhookID := "whk-123"
	targetURL := "https://hooks.superplane.dev/rootly"
	deterministicName := rootlyWebhookName(webhookID)

	t.Run("prefers deterministic name match", func(t *testing.T) {
		endpoints := []WebhookEndpoint{
			{ID: "1", Name: "SuperPlane", URL: targetURL},
			{ID: "2", Name: deterministicName, URL: "https://other.example.com"},
		}

		selected := selectWebhookEndpoint(endpoints, deterministicName, targetURL)
		require.NotNil(t, selected)
		assert.Equal(t, "2", selected.ID)
	})

	t.Run("falls back to legacy name on URL match", func(t *testing.T) {
		endpoints := []WebhookEndpoint{
			{ID: "legacy", Name: rootlyLegacyWebhookName, URL: targetURL},
			{ID: "other", Name: "Other", URL: targetURL},
		}

		selected := selectWebhookEndpoint(endpoints, deterministicName, targetURL)
		require.NotNil(t, selected)
		assert.Equal(t, "legacy", selected.ID)
	})

	t.Run("falls back to first URL match when no name matches", func(t *testing.T) {
		endpoints := []WebhookEndpoint{
			{ID: "a", Name: "Alpha", URL: targetURL},
			{ID: "b", Name: "Beta", URL: targetURL},
		}

		selected := selectWebhookEndpoint(endpoints, deterministicName, targetURL)
		require.NotNil(t, selected)
		assert.Equal(t, "a", selected.ID)
	})

	t.Run("returns nil when no URL or name matches", func(t *testing.T) {
		endpoints := []WebhookEndpoint{
			{ID: "a", Name: "Alpha", URL: "https://example.com"},
		}

		selected := selectWebhookEndpoint(endpoints, deterministicName, targetURL)
		assert.Nil(t, selected)
	})
}
