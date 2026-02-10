package railway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__RailwayWebhookHandler__CompareConfig(t *testing.T) {
	handler := &RailwayWebhookHandler{}

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
				Project: "proj-123",
			},
			configB: WebhookConfiguration{
				Project: "proj-123",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different projects",
			configA: WebhookConfiguration{
				Project: "proj-123",
			},
			configB: WebhookConfiguration{
				Project: "proj-456",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"project": "proj-123",
			},
			configB: map[string]any{
				"project": "proj-123",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "map representations with different projects",
			configA: map[string]any{
				"project": "proj-123",
			},
			configB: map[string]any{
				"project": "proj-456",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				Project: "proj-123",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Project: "proj-123",
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
		{
			name:        "both configurations invalid",
			configA:     "invalid",
			configB:     123,
			expectEqual: false,
			expectError: true,
		},
		{
			name: "empty project strings are equal",
			configA: WebhookConfiguration{
				Project: "",
			},
			configB: WebhookConfiguration{
				Project: "",
			},
			expectEqual: true,
			expectError: false,
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

func Test__RailwayWebhookHandler__Setup(t *testing.T) {
	t.Run("returns metadata with project ID", func(t *testing.T) {
		handler := &RailwayWebhookHandler{}
		webhookCtx := &mockWebhookContext{
			configuration: WebhookConfiguration{
				Project: "proj-123",
			},
		}

		ctx := core.WebhookHandlerContext{
			Webhook: webhookCtx,
		}

		metadata, err := handler.Setup(ctx)
		require.NoError(t, err)

		webhookMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "proj-123", webhookMetadata.Project)
	})

	t.Run("returns metadata from map configuration", func(t *testing.T) {
		handler := &RailwayWebhookHandler{}
		webhookCtx := &mockWebhookContext{
			configuration: map[string]any{
				"project": "proj-456",
			},
		}

		ctx := core.WebhookHandlerContext{
			Webhook: webhookCtx,
		}

		metadata, err := handler.Setup(ctx)
		require.NoError(t, err)

		webhookMetadata, ok := metadata.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "proj-456", webhookMetadata.Project)
	})
}

func Test__RailwayWebhookHandler__Cleanup(t *testing.T) {
	t.Run("cleanup returns no error", func(t *testing.T) {
		handler := &RailwayWebhookHandler{}
		ctx := core.WebhookHandlerContext{}
		err := handler.Cleanup(ctx)
		assert.NoError(t, err)
	})
}

// Mock implementation of core.WebhookContext for testing

type mockWebhookContext struct {
	configuration any
}

func (m *mockWebhookContext) GetID() string {
	return "webhook-123"
}

func (m *mockWebhookContext) GetURL() string {
	return "https://example.com/webhook"
}

func (m *mockWebhookContext) GetConfiguration() any {
	return m.configuration
}

func (m *mockWebhookContext) GetMetadata() any {
	return nil
}

func (m *mockWebhookContext) GetSecret() ([]byte, error) {
	return nil, nil
}

func (m *mockWebhookContext) SetSecret(secret []byte) error {
	return nil
}
