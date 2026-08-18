package runner

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestValidateComputeSelection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		enableServerless bool
		functionType     string
		machineType      string
		wantErr          bool
	}{
		{name: "machine type selected", machineType: testRunnerMachineType},
		{name: "machine type missing", wantErr: true},
		{name: "serverless function type selected", enableServerless: true, functionType: FunctionType1GB},
		{name: "serverless ignores missing machine type", enableServerless: true, functionType: FunctionType512MB},
		{name: "serverless function type padded", enableServerless: true, functionType: "  4gb  "},
		{name: "serverless function type missing", enableServerless: true, wantErr: true},
		{name: "serverless function type unknown", enableServerless: true, functionType: "8gb", wantErr: true},
		{name: "serverless ignores machine type only", enableServerless: true, machineType: testRunnerMachineType, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateComputeSelection(tc.enableServerless, tc.functionType, tc.machineType)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestResolvedFunctionType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, FunctionType2GB, resolvedFunctionType(true, "  2gb  "))
	assert.Empty(t, resolvedFunctionType(false, FunctionType2GB))
	assert.Empty(t, resolvedFunctionType(true, "   "))
}

func TestSetComputeTargetServerless(t *testing.T) {
	t.Parallel()

	var req brokerCreateTaskRequest
	require.NoError(t, req.setComputeTarget(CreateTaskParams{
		FunctionType:  FunctionType1GB,
		MachineType:   testRunnerMachineType,
		ExecutionMode: ExecutionModeDocker,
		DockerImage:   "debian:bookworm-slim",
	}))

	assert.Equal(t, FunctionType1GB, req.FunctionType)
	assert.Empty(t, req.FleetID)
	assert.Empty(t, req.ExecutionMode)
	assert.Empty(t, req.DockerImage)

	body, err := json.Marshal(req)
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	assert.Equal(t, FunctionType1GB, sent["function_type"])
	assert.NotContains(t, sent, "fleet_id")
	assert.NotContains(t, sent, "execution_mode")
	assert.NotContains(t, sent, "docker_image")
}

func TestSetComputeTargetMachine(t *testing.T) {
	t.Parallel()

	var req brokerCreateTaskRequest
	require.NoError(t, req.setComputeTarget(CreateTaskParams{
		MachineType:   testRunnerMachineType,
		ExecutionMode: " Docker ",
		DockerImage:   " debian:bookworm-slim ",
	}))

	assert.Equal(t, testRunnerMachineType, req.FleetID)
	assert.Equal(t, ExecutionModeDocker, req.ExecutionMode)
	assert.Equal(t, "debian:bookworm-slim", req.DockerImage)
	assert.Empty(t, req.FunctionType)

	body, err := json.Marshal(req)
	require.NoError(t, err)

	var sent map[string]any
	require.NoError(t, json.Unmarshal(body, &sent))
	assert.Equal(t, testRunnerMachineType, sent["fleet_id"])
	assert.NotContains(t, sent, "function_type")
}

func TestSetComputeTargetDefaultsExecutionModeToHost(t *testing.T) {
	t.Parallel()

	var req brokerCreateTaskRequest
	require.NoError(t, req.setComputeTarget(CreateTaskParams{MachineType: testRunnerMachineType}))
	assert.Equal(t, ExecutionModeHost, req.ExecutionMode)
}

func TestSetComputeTargetRequiresComputeTarget(t *testing.T) {
	t.Parallel()

	var req brokerCreateTaskRequest
	require.Error(t, req.setComputeTarget(CreateTaskParams{}))
}

func TestValidateConfigurationServerlessOmitsMachineType(t *testing.T) {
	t.Parallel()

	serverless := map[string]any{
		"enableServerless": true,
		"functionType":     FunctionType512MB,
		"commands":         "echo hi",
	}
	require.NoError(t, configuration.ValidateConfiguration((&Runner{}).Configuration(), serverless))

	spec, err := decodeRunnerSpec(serverless)
	require.NoError(t, err)
	require.True(t, spec.EnableServerless)
	require.Equal(t, FunctionType512MB, spec.FunctionType)
	require.NoError(t, validateRunnerSpec(spec))
}

func TestValidateConfigurationRequiresMachineTypeWhenNotServerless(t *testing.T) {
	t.Parallel()

	err := configuration.ValidateConfiguration((&Runner{}).Configuration(), map[string]any{
		"enableServerless": false,
		"commands":         "echo hi",
	})
	require.Error(t, err)
}

func TestValidateConfigurationRequiresFunctionTypeWhenServerless(t *testing.T) {
	t.Parallel()

	err := configuration.ValidateConfiguration((&Runner{}).Configuration(), map[string]any{
		"enableServerless": true,
		"commands":         "echo hi",
	})
	require.Error(t, err)
}

func TestServerlessSpecsIgnoreDockerConfiguration(t *testing.T) {
	t.Parallel()

	script := `echo ok > "$SUPERPLANE_RESULT_FILE"`

	require.NoError(t, validateRunnerSpec(Spec{
		EnableServerless: true,
		FunctionType:     FunctionType512MB,
		Commands:         "echo hi",
		ExecutionMode:    ExecutionModeDocker,
	}))
	require.NoError(t, validateRunBashSpec(RunBashSpec{
		EnableServerless: true,
		FunctionType:     FunctionType512MB,
		Script:           script,
		ExecutionMode:    ExecutionModeDocker,
	}))
	require.NoError(t, validateRunJSSpec(RunJSSpec{
		EnableServerless: true,
		FunctionType:     FunctionType512MB,
		Script:           script,
		ExecutionMode:    ExecutionModeDocker,
	}))
	require.NoError(t, validateRunPythonSpec(RunPythonSpec{
		EnableServerless: true,
		FunctionType:     FunctionType512MB,
		Script:           script,
		ExecutionMode:    ExecutionModeDocker,
	}))

	assert.Empty(t, resolvedDockerImageRef(Spec{
		EnableServerless:  true,
		ExecutionMode:     ExecutionModeDocker,
		DockerImagePreset: "debian:bookworm-slim",
	}))
}

func TestServerlessSpecsRequireFunctionType(t *testing.T) {
	t.Parallel()

	script := `echo ok > "$SUPERPLANE_RESULT_FILE"`

	require.Error(t, validateRunnerSpec(Spec{EnableServerless: true, Commands: "echo hi"}))
	require.Error(t, validateRunBashSpec(RunBashSpec{EnableServerless: true, Script: script}))
	require.Error(t, validateRunJSSpec(RunJSSpec{EnableServerless: true, Script: script}))
	require.Error(t, validateRunPythonSpec(RunPythonSpec{EnableServerless: true, Script: script}))
}
