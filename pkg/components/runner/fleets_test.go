package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireMachineType(t *testing.T) {
	t.Parallel()
	got, err := requireMachineType(MachineTypeE1LargeAMD64)
	require.NoError(t, err)
	assert.Equal(t, MachineTypeE1LargeAMD64, got)

	_, err = requireMachineType("")
	require.Error(t, err)
}

func TestResolveCreateTaskFleetID(t *testing.T) {
	t.Run("uses machine type when override is empty", func(t *testing.T) {
		t.Setenv("TASK_BROKER_FLEET_ID", "")
		got, err := resolveCreateTaskFleetID(MachineTypeE1LargeARM64)
		require.NoError(t, err)
		assert.Equal(t, MachineTypeE1LargeARM64, got)
	})

	t.Run("overrides machine type when TASK_BROKER_FLEET_ID is set", func(t *testing.T) {
		t.Setenv("TASK_BROKER_FLEET_ID", " local ")
		got, err := resolveCreateTaskFleetID(MachineTypeE1LargeAMD64)
		require.NoError(t, err)
		assert.Equal(t, "local", got)
	})

	t.Run("rejects empty machine type without override", func(t *testing.T) {
		t.Setenv("TASK_BROKER_FLEET_ID", "")
		_, err := resolveCreateTaskFleetID("")
		require.Error(t, err)
	})
}
