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

func TestResolveBrokerFleetIDUsesEnvOverride(t *testing.T) {
	t.Setenv("TASK_BROKER_FLEET_ID", "")
	got, err := resolveBrokerFleetID(MachineTypeE1LargeAMD64)
	require.NoError(t, err)
	assert.Equal(t, MachineTypeE1LargeAMD64, got)

	t.Setenv("TASK_BROKER_FLEET_ID", "aws-standard-1")
	got, err = resolveBrokerFleetID(MachineTypeE1LargeAMD64)
	require.NoError(t, err)
	assert.Equal(t, "aws-standard-1", got)

	_, err = resolveBrokerFleetID("")
	require.Error(t, err)
}
