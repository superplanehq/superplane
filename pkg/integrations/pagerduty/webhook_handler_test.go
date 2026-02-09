package pagerduty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__PagerDutyWebhookHandler__CompareConfig(t *testing.T) {
	handler := &PagerDutyWebhookHandler{}

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
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different service",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service2",
				},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different event",
			configA: WebhookConfiguration{
				Events: []string{"incident.resolved"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "subset of events",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered", "incident.resolved"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"events": []string{"incident.triggered"},
				"filter": map[string]string{
					"type": "service_reference",
					"id":   "service1",
				},
			},
			configB: map[string]any{
				"events": []string{"incident.triggered"},
				"filter": map[string]string{
					"type": "service_reference",
					"id":   "service1",
				},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered"},
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

			assert.Equal(t, tc.expectEqual, equal, "expected config to be equal, but they are different")
		})
	}
}
