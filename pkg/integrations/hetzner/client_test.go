package hetzner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__resolveServerID(t *testing.T) {
	t.Run("valid string id", func(t *testing.T) {
		id, err := resolveServerID(map[string]any{"server": "12345"})
		require.NoError(t, err)
		assert.Equal(t, "12345", id)
	})

	t.Run("valid numeric id", func(t *testing.T) {
		id, err := resolveServerID(map[string]any{"server": 12345.0})
		require.NoError(t, err)
		assert.Equal(t, "12345", id)
	})

	t.Run("nil placeholder id is rejected", func(t *testing.T) {
		_, err := resolveServerID(map[string]any{"server": "<nil>"})
		require.Error(t, err)
		assert.Equal(t, "server is required", err.Error())
	})

	t.Run("template placeholder id is rejected", func(t *testing.T) {
		_, err := resolveServerID(map[string]any{"server": "{{ steps.create_server.id }}"})
		require.Error(t, err)
		assert.Equal(t, "server is required", err.Error())
	})
}

func Test__resolveLoadBalancerID(t *testing.T) {
	t.Run("nil placeholder id is rejected", func(t *testing.T) {
		_, err := resolveLoadBalancerID(map[string]any{"loadBalancer": "nil"})
		require.Error(t, err)
		assert.Equal(t, "loadBalancer is required", err.Error())
	})
}

func Test__resolveImageID(t *testing.T) {
	t.Run("null placeholder id is rejected", func(t *testing.T) {
		_, err := resolveImageID(map[string]any{"snapshot": "null"}, "snapshot")
		require.Error(t, err)
		assert.Equal(t, "snapshot is required", err.Error())
	})
}
