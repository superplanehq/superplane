package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestDecodeRunJSSpecDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunJSSpec(map[string]any{
		"machine_type": testRunnerMachineType,
		"script":       "function main() { return null; }",
	})
	require.NoError(t, err)
	require.Equal(t, ExecutionModeHost, spec.ExecutionMode)
	require.Equal(t, DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
	require.Equal(t, runJSDefaultDockerPreset, spec.DockerImagePreset)
}

func TestValidateConfigurationRunJSScriptDefault(t *testing.T) {
	t.Parallel()

	r := &RunJS{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type": testRunnerMachineType,
		"script":       defaultRunJSScript,
	})
	require.NoError(t, err)
}

func TestValidateConfigurationRunJSSetupCommands(t *testing.T) {
	t.Parallel()

	r := &RunJS{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":          testRunnerMachineType,
		"script":                defaultRunJSScript,
		"enable_setup_commands": true,
		"setup_commands":        "npm ci",
	})
	require.NoError(t, err)
}

func TestValidateConfigurationRunJSDockerDefaults(t *testing.T) {
	t.Parallel()

	r := &RunJS{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":              testRunnerMachineType,
		"execution_mode":            ExecutionModeDocker,
		"script":                    "function main() { return 1; }",
		"execution_timeout_seconds": 0,
	})
	require.NoError(t, err)
}
