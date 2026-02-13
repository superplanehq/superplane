package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__LinearWebhookHandler__CompareConfig(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("identical team and resource types", func(t *testing.T) {
		a := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Issue"}}
		b := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Issue"}}
		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different team", func(t *testing.T) {
		a := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Issue"}}
		b := WebhookConfiguration{TeamID: "t2", ResourceTypes: []string{"Issue"}}
		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different AllPublicTeams", func(t *testing.T) {
		a := WebhookConfiguration{AllPublicTeams: true, ResourceTypes: []string{"Issue"}}
		b := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Issue"}}
		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different resource types", func(t *testing.T) {
		a := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Issue"}}
		b := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Comment"}}
		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("map representations", func(t *testing.T) {
		a := map[string]any{"teamId": "t1", "resourceTypes": []string{"Issue"}}
		b := WebhookConfiguration{TeamID: "t1", ResourceTypes: []string{"Issue"}}
		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.True(t, equal)
	})
}
