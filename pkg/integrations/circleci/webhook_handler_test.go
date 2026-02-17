package circleci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__CircleCIWebhookHandler__CompareConfig(t *testing.T) {
	h := &CircleCIWebhookHandler{}

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
		{
			name: "same events - should match",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "configA has superset of events - should match (webhook reuse)",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed", "job-completed"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "configA has subset of events - should NOT match",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed", "job-completed"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different events - should NOT match",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"job-completed"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "empty events in configB - should match (no events needed)",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "empty events in configA - should match (normalized to default)",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "invalid event type in configA - should error",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"invalid-event"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed"},
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "duplicate events - should dedupe and match",
			configA: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed", "workflow-completed", "job-completed"},
			},
			configB: WebhookConfiguration{
				ProjectSlug: "gh/username/repo",
				Events:      []string{"workflow-completed", "job-completed"},
			},
			expectEqual: true,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := h.CompareConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}
