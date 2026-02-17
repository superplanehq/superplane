package createvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateCreateVMConfig(t *testing.T) {
	t.Run("valid config returns ok", func(t *testing.T) {
		config := CreateVMConfig{
			InstanceName: "my-vm-01",
			Zone:         "us-central1-a",
			MachineType:  "e2-medium",
		}
		msg, ok := validateCreateVMConfig(config)
		require.True(t, ok)
		assert.Empty(t, msg)
	})

	t.Run("empty instance name returns error", func(t *testing.T) {
		config := CreateVMConfig{Zone: "us-central1-a", MachineType: "e2-medium"}
		msg, ok := validateCreateVMConfig(config)
		require.False(t, ok)
		assert.Equal(t, "instance name is required", msg)
	})

	t.Run("instance name must match GCP pattern", func(t *testing.T) {
		for _, name := range []string{"My-VM", "123vm", "vm_01", "-vm", "vm-"} {
			config := CreateVMConfig{InstanceName: name, Zone: "us-central1-a", MachineType: "e2-medium"}
			msg, ok := validateCreateVMConfig(config)
			require.False(t, ok, "expected invalid for %q", name)
			assert.Contains(t, msg, "instance name must be")
		}
	})

	t.Run("valid instance names", func(t *testing.T) {
		for _, name := range []string{"my-vm", "my-vm-01", "a1", "z"} {
			config := CreateVMConfig{InstanceName: name, Zone: "us-central1-a", MachineType: "e2-medium"}
			_, ok := validateCreateVMConfig(config)
			require.True(t, ok, "expected valid for %q", name)
		}
	})

	t.Run("empty zone returns error", func(t *testing.T) {
		config := CreateVMConfig{InstanceName: "my-vm", Zone: "  ", MachineType: "e2-medium"}
		msg, ok := validateCreateVMConfig(config)
		require.False(t, ok)
		assert.Equal(t, "zone is required", msg)
	})

	t.Run("empty machine type returns error", func(t *testing.T) {
		config := CreateVMConfig{InstanceName: "my-vm", Zone: "us-central1-a", MachineType: ""}
		msg, ok := validateCreateVMConfig(config)
		require.False(t, ok)
		assert.Equal(t, "machine type is required", msg)
	})
}
