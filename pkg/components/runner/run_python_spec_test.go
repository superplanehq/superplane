package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestDecodeRunPythonSpecDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunPythonSpec(map[string]any{
		"machine_type": testRunnerMachineType,
		"script":       "def main(payload):\n    return None",
	})
	require.NoError(t, err)
	require.Equal(t, ExecutionModeHost, spec.ExecutionMode)
	require.Equal(t, DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
	require.Equal(t, runPythonDefaultDockerPreset, spec.DockerImagePreset)
}

func TestValidateConfigurationRunPythonScriptDefault(t *testing.T) {
	t.Parallel()

	r := &RunPython{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type": testRunnerMachineType,
		"script":       defaultRunPythonScript,
	})
	require.NoError(t, err)
}

func TestValidateConfigurationRunPythonSetupCommands(t *testing.T) {
	t.Parallel()

	r := &RunPython{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":          testRunnerMachineType,
		"script":                defaultRunPythonScript,
		"enable_setup_commands": true,
		"setup_commands":        "pip install requests",
	})
	require.NoError(t, err)
}

func TestValidateConfigurationRunPythonDockerDefaults(t *testing.T) {
	t.Parallel()

	r := &RunPython{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":              testRunnerMachineType,
		"execution_mode":            ExecutionModeDocker,
		"script":                    "def main(payload):\n    return 1",
		"execution_timeout_seconds": 0,
	})
	require.NoError(t, err)
}
