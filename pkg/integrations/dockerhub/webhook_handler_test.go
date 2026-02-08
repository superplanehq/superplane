package dockerhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__DockerHubWebhookHandler__CompareConfig(t *testing.T) {
	handler := &DockerHubWebhookHandler{}

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
				Namespace:  "superplane",
				Repository: "demo",
			},
			configB: WebhookConfiguration{
				Namespace:  "superplane",
				Repository: "demo",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different repositories",
			configA: WebhookConfiguration{
				Namespace:  "superplane",
				Repository: "demo",
			},
			configB: WebhookConfiguration{
				Namespace:  "superplane",
				Repository: "other",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different namespaces",
			configA: WebhookConfiguration{
				Namespace:  "superplane",
				Repository: "demo",
			},
			configB: WebhookConfiguration{
				Namespace:  "org",
				Repository: "demo",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "map config comparison",
			configA: map[string]any{
				"namespace":  "superplane",
				"repository": "demo",
			},
			configB: map[string]any{
				"namespace":  "superplane",
				"repository": "demo",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:        "invalid first config",
			configA:     "invalid",
			configB:     WebhookConfiguration{Namespace: "superplane", Repository: "demo"},
			expectEqual: false,
			expectError: true,
		},
		{
			name:        "invalid second config",
			configA:     WebhookConfiguration{Namespace: "superplane", Repository: "demo"},
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
