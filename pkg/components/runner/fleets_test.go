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
