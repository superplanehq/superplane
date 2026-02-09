package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__GitLabWebhookHandler__CompareConfig(t *testing.T) {
	handler := &GitLabWebhookHandler{}

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
				ProjectID: "123",
				EventType: "push",
			},
			configB: WebhookConfiguration{
				ProjectID: "123",
				EventType: "push",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different project IDs",
			configA: WebhookConfiguration{
				ProjectID: "123",
				EventType: "push",
			},
			configB: WebhookConfiguration{
				ProjectID: "456",
				EventType: "push",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different event types",
			configA: WebhookConfiguration{
				ProjectID: "123",
				EventType: "push",
			},
			configB: WebhookConfiguration{
				ProjectID: "123",
				EventType: "merge_requests",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "both fields different",
			configA: WebhookConfiguration{
				ProjectID: "123",
				EventType: "push",
			},
			configB: WebhookConfiguration{
				ProjectID: "456",
				EventType: "issues",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"projectId": "123",
				"eventType": "push",
			},
			configB: map[string]any{
				"projectId": "123",
				"eventType": "push",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				ProjectID: "123",
				EventType: "push",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				ProjectID: "123",
				EventType: "push",
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
