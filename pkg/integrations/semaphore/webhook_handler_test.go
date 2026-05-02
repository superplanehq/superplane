package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore/common"
)

func Test__SemaphoreWebhookHandler__CompareConfig(t *testing.T) {
	handler := &SemaphoreWebhookHandler{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical configurations",
			configA: common.WebhookConfiguration{
				Project: "my-project",
			},
			configB: common.WebhookConfiguration{
				Project: "my-project",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different projects",
			configA: common.WebhookConfiguration{
				Project: "my-project",
			},
			configB: common.WebhookConfiguration{
				Project: "other-project",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"project": "my-project",
			},
			configB: map[string]any{
				"project": "my-project",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: common.WebhookConfiguration{
				Project: "my-project",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: common.WebhookConfiguration{
				Project: "my-project",
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
