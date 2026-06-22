package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestDecodeRunBashSpecDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunBashSpec(map[string]any{
		"machine_type": testRunnerMachineType,
		"script":       `echo ok > "$SUPERPLANE_RESULT_FILE"`,
	})
	require.NoError(t, err)
	require.Equal(t, ExecutionModeHost, spec.ExecutionMode)
	require.Equal(t, DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
	require.Equal(t, runBashDefaultDockerPreset, spec.DockerImagePreset)
}

func TestValidateConfigurationRunBashScriptDefault(t *testing.T) {
	t.Parallel()

	r := &RunBash{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type": testRunnerMachineType,
		"script":       defaultRunBashScript,
	})
	require.NoError(t, err)
}

func TestValidateConfigurationRunBashSetupCommands(t *testing.T) {
	t.Parallel()

	r := &RunBash{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":          testRunnerMachineType,
		"script":                defaultRunBashScript,
		"enable_setup_commands": true,
		"setup_commands":        "apt-get update",
	})
	require.NoError(t, err)
}

func TestValidateConfigurationRunBashDockerDefaults(t *testing.T) {
	t.Parallel()

	r := &RunBash{}
	err := configuration.ValidateConfiguration(r.Configuration(), map[string]any{
		"machine_type":              testRunnerMachineType,
		"execution_mode":            ExecutionModeDocker,
		"script":                    `echo ok > "$SUPERPLANE_RESULT_FILE"`,
		"execution_timeout_seconds": 0,
	})
	require.NoError(t, err)
}
