package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMachineTypeLabel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, MachineTypeE1LargeAMD64, MachineTypeLabel("aws-standard-1"))
	assert.Equal(t, MachineTypeE1LargeARM64, MachineTypeLabel("aws-arm64-1"))
	assert.Equal(t, MachineTypeE1TinyAMD64, MachineTypeLabel("e1-tiny-amd64"))
	assert.Equal(t, MachineTypeE1TinyARM64, MachineTypeLabel("e1-tiny-arm64"))
	assert.Equal(t, "custom-fleet", MachineTypeLabel("custom-fleet"))
}

func TestRequireMachineType(t *testing.T) {
	t.Parallel()
	got, err := requireMachineType("aws-standard-1")
	require.NoError(t, err)
	assert.Equal(t, "aws-standard-1", got)

	_, err = requireMachineType("")
	require.Error(t, err)
}
