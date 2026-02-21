package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
