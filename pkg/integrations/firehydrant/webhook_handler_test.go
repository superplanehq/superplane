package firehydrant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__FireHydrantWebhookHandler__CompareConfig(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

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
				Events: []string{"incident_created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident_created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different events",
			configA: WebhookConfiguration{
				Events: []string{"incident_created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident_updated"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "superset of events (A contains all of B)",
			configA: WebhookConfiguration{
				Events: []string{"incident_created", "incident_updated"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident_created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "subset of events (A does not contain all of B)",
			configA: WebhookConfiguration{
				Events: []string{"incident_created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident_created", "incident_updated"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"events": []string{"incident_created", "incident_updated"},
			},
			configB: map[string]any{
				"events": []string{"incident_created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				Events: []string{"incident_created"},
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Events: []string{"incident_created"},
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

func Test__FireHydrantWebhookHandler__Merge(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

	t.Run("merges new events", func(t *testing.T) {
		current := WebhookConfiguration{
			Events: []string{"incident_created"},
		}

		requested := WebhookConfiguration{
			Events: []string{"incident_updated", "incident_created"},
		}

		merged, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)

		result := merged.(WebhookConfiguration)
		assert.ElementsMatch(t, []string{"incident_created", "incident_updated"}, result.Events)
		assert.True(t, changed)
	})

	t.Run("no change when all events already present", func(t *testing.T) {
		current := WebhookConfiguration{
			Events: []string{"incident_created"},
		}

		requested := WebhookConfiguration{
			Events: []string{"incident_created"},
		}

		_, changed, err := handler.Merge(current, requested)
		require.NoError(t, err)
		assert.False(t, changed)
	})
}
