package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__OpenAI__CapabilityMapper__PermissionSet(t *testing.T) {
	mapper := NewCapabilityMapper()

	t.Run("baseline includes models read", func(t *testing.T) {
		permissions := mapper.PermissionSet([]string{}, true)
		require.Contains(t, permissions, PermissionEndpointModels)
		assert.Equal(t, PermissionAccessRead, permissions[PermissionEndpointModels])
	})

	t.Run("text prompt requires responses write", func(t *testing.T) {
		permissions := mapper.PermissionSet([]string{"openai.textPrompt"}, true)
		require.Contains(t, permissions, PermissionEndpointResponses)
		assert.Equal(t, PermissionAccessWrite, permissions[PermissionEndpointResponses])
	})
}

func Test__OpenAI__CapabilityMapper__FindPermissionUpdates(t *testing.T) {
	t.Run("ignores already satisfied permissions", func(t *testing.T) {
		existing := PermissionSet{
			PermissionEndpointModels:    PermissionAccessRead,
			PermissionEndpointResponses: PermissionAccessWrite,
		}

		requested := PermissionSet{
			PermissionEndpointResponses: PermissionAccessWrite,
		}

		diff := FindPermissionUpdates(existing, requested)
		assert.True(t, diff.IsEmpty())
	})

	t.Run("returns missing write permission", func(t *testing.T) {
		existing := PermissionSet{
			PermissionEndpointModels: PermissionAccessRead,
		}

		requested := PermissionSet{
			PermissionEndpointResponses: PermissionAccessWrite,
		}

		diff := FindPermissionUpdates(existing, requested)
		require.Contains(t, diff, PermissionEndpointResponses)
		assert.Equal(t, PermissionAccessWrite, diff[PermissionEndpointResponses])
	})
}
