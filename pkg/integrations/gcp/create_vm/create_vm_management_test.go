package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BuildInstanceMetadata(t *testing.T) {
	t.Run("empty config returns nil", func(t *testing.T) {
		out := BuildInstanceMetadata(ManagementConfig{})
		assert.Nil(t, out)
	})
	t.Run("startup and shutdown script", func(t *testing.T) {
		cfg := ManagementConfig{
			StartupScript:  " echo hello ",
			ShutdownScript: " echo bye ",
		}
		out := BuildInstanceMetadata(cfg)
		require.NotNil(t, out)
		require.Len(t, out.Items, 2)
		keys := make(map[string]string)
		for _, it := range out.Items {
			keys[it.Key] = *it.Value
		}
		assert.Equal(t, "echo hello", keys["startup-script"])
		assert.Equal(t, "echo bye", keys["shutdown-script"])
	})
	t.Run("custom metadata items", func(t *testing.T) {
		cfg := ManagementConfig{
			MetadataItems: []MetadataKeyValue{
				{Key: "env", Value: "prod"},
				{Key: "team", Value: "platform"},
			},
		}
		out := BuildInstanceMetadata(cfg)
		require.NotNil(t, out)
		require.Len(t, out.Items, 2)
		keys := make(map[string]string)
		for _, it := range out.Items {
			keys[it.Key] = *it.Value
		}
		assert.Equal(t, "prod", keys["env"])
		assert.Equal(t, "platform", keys["team"])
	})
	t.Run("skips empty key and dedupes", func(t *testing.T) {
		cfg := ManagementConfig{
			MetadataItems: []MetadataKeyValue{
				{Key: "", Value: "v"},
				{Key: "k", Value: "first"},
				{Key: "k", Value: "second"},
			},
		}
		out := BuildInstanceMetadata(cfg)
		require.NotNil(t, out)
		require.Len(t, out.Items, 1)
		assert.Equal(t, "first", *out.Items[0].Value)
	})
}

func Test_BuildScheduling(t *testing.T) {
	t.Run("default automatic restart and migrate", func(t *testing.T) {
		out := BuildScheduling(ManagementConfig{})
		require.NotNil(t, out)
		require.NotNil(t, out.AutomaticRestart)
		assert.True(t, *out.AutomaticRestart)
		assert.Equal(t, OnHostMaintenanceMigrate, out.OnHostMaintenance)
	})
	t.Run("explicit automatic restart false", func(t *testing.T) {
		f := false
		out := BuildScheduling(ManagementConfig{AutomaticRestart: &f})
		require.NotNil(t, out)
		require.NotNil(t, out.AutomaticRestart)
		assert.False(t, *out.AutomaticRestart)
	})
	t.Run("on host maintenance terminate", func(t *testing.T) {
		out := BuildScheduling(ManagementConfig{OnHostMaintenance: OnHostMaintenanceTerminate})
		require.NotNil(t, out)
		assert.Equal(t, OnHostMaintenanceTerminate, out.OnHostMaintenance)
	})
}
