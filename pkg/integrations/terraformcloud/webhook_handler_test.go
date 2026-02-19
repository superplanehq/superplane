package terraformcloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__WebhookHandler__CompareConfig(t *testing.T) {
	handler := &TerraformCloudWebhookHandler{}

	t.Run("same workspace and triggers -> match", func(t *testing.T) {
		a := WebhookConfiguration{
			WorkspaceID: "ws-123",
			Triggers:    []string{"run:completed", "run:errored"},
		}

		b := WebhookConfiguration{
			WorkspaceID: "ws-123",
			Triggers:    []string{"run:completed"},
		}

		match, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("different workspace -> no match", func(t *testing.T) {
		a := WebhookConfiguration{
			WorkspaceID: "ws-123",
			Triggers:    []string{"run:completed"},
		}

		b := WebhookConfiguration{
			WorkspaceID: "ws-456",
			Triggers:    []string{"run:completed"},
		}

		match, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("requested triggers not in current -> no match", func(t *testing.T) {
		a := WebhookConfiguration{
			WorkspaceID: "ws-123",
			Triggers:    []string{"run:completed"},
		}

		b := WebhookConfiguration{
			WorkspaceID: "ws-123",
			Triggers:    []string{"run:errored"},
		}

		match, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, match)
	})
}

func Test__NormalizeTriggers(t *testing.T) {
	t.Run("empty triggers returns defaults", func(t *testing.T) {
		result, err := normalizeTriggers([]string{})
		require.NoError(t, err)
		assert.Equal(t, defaultTriggers, result)
	})

	t.Run("duplicates are removed", func(t *testing.T) {
		result, err := normalizeTriggers([]string{"run:completed", "run:completed"})
		require.NoError(t, err)
		assert.Equal(t, []string{"run:completed"}, result)
	})

	t.Run("unsupported trigger returns error", func(t *testing.T) {
		_, err := normalizeTriggers([]string{"run:unknown"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported Terraform Cloud trigger")
	})
}
