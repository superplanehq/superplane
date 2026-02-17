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

	testCases := []struct {
		name           string
		current        any
		requested      any
		expectedEvents []string
		expectChanged  bool
		expectError    bool
	}{
		{
			name: "adds new event types",
			current: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			requested: WebhookConfiguration{
				Events: []string{"incident.updated"},
			},
			expectedEvents: []string{"incident.created", "incident.updated"},
			expectChanged:  true,
			expectError:    false,
		},
		{
			name: "keeps existing when already present",
			current: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			requested: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectedEvents: []string{"incident.created"},
			expectChanged:  false,
			expectError:    false,
		},
		{
			name:    "invalid current config",
			current: 123,
			requested: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectedEvents: nil,
			expectChanged:  false,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			merged, changed, err := handler.Merge(tc.current, tc.requested)
			if tc.expectError {
				assert.Error(t, err, "expected error, but got none")
				return
			}

			require.NoError(t, err, "did not expect, but got an error")
			assert.Equal(t, tc.expectChanged, changed)

			config := WebhookConfiguration{}
			err = mapstructure.Decode(merged, &config)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedEvents, config.Events)

			if !tc.expectChanged && !tc.expectError {
				assert.Equal(t, tc.current, merged, "expected Merge to return current when unchanged")
			}
		})
	}
}
