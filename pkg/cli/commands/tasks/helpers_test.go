package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestResolveTaskID(t *testing.T) {
	t.Run("returns the trimmed identifier", func(t *testing.T) {
		resolved, err := resolveTaskID("  1823  ")
		require.NoError(t, err)
		assert.Equal(t, "1823", resolved)
	})

	t.Run("accepts a key", func(t *testing.T) {
		resolved, err := resolveTaskID("SUPER-1823")
		require.NoError(t, err)
		assert.Equal(t, "SUPER-1823", resolved)
	})

	t.Run("requires a value", func(t *testing.T) {
		_, err := resolveTaskID("   ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--task is required")
	})
}

func TestTaskDisplayID(t *testing.T) {
	task := openapi_client.NewFactoriesWorkOrder()
	task.SetId("22222222-2222-2222-2222-222222222222")
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", taskDisplayID(*task))

	task.SetKey("SUPER-1823")
	assert.Equal(t, "SUPER-1823", taskDisplayID(*task))

	task.SetNumber("1823")
	assert.Equal(t, "1823", taskDisplayID(*task))
}
