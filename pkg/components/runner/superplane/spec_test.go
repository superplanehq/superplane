package superplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestComponentNameMatchesModels(t *testing.T) {
	t.Parallel()
	assert.Equal(t, models.SuperPlaneRunnerComponent, ComponentName)
}

func TestValidateRunSuperPlaneSpecRejectsUnsupportedFields(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	base := map[string]any{
		"machineType": runner.MachineTypeE1LargeAMD64,
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": prompt},
		},
	}

	_, err := decodeRunSuperPlaneSpec(base)
	require.NoError(t, err)

	for _, key := range []string{"credentials", "maxTurns"} {
		cfg := cloneConfig(base)
		cfg[key] = "not-allowed"
		_, err := decodeRunSuperPlaneSpec(cfg)
		require.Error(t, err, key)
		assert.Contains(t, err.Error(), key)
	}
}

func TestValidateRunSuperPlaneSpecAcceptsOptionalModel(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	cfg := map[string]any{
		"machineType": runner.MachineTypeE1LargeAMD64,
		"model":       "openrouter::anthropic/claude-sonnet-4-6",
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": prompt},
		},
	}

	spec, err := decodeRunSuperPlaneSpec(cfg)
	require.NoError(t, err)
	require.NoError(t, validateRunSuperPlaneSpec(spec))
	assert.Equal(t, "openrouter::anthropic/claude-sonnet-4-6", spec.Model)

	cfg["model"] = "hosted::anthropic::claude-sonnet-4-6"
	spec, err = decodeRunSuperPlaneSpec(cfg)
	require.NoError(t, err)
	require.NoError(t, validateRunSuperPlaneSpec(spec))
	selected, ok, err := specSelectedSuperPlaneModel(spec)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "anthropic", selected.Provider)
	assert.Equal(t, "claude-sonnet-4-6", selected.Model)
}

func TestValidateRunSuperPlaneSpecRejectsInvalidModel(t *testing.T) {
	t.Parallel()

	spec := RunSuperPlaneSpec{
		MachineType: runner.MachineTypeE1LargeAMD64,
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: strPtr("fix tests")},
		},
		Model: "not-a-hosted-model",
	}
	err := validateRunSuperPlaneSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestValidateRunSuperPlaneSpecRequiresPrompt(t *testing.T) {
	t.Parallel()

	spec := RunSuperPlaneSpec{
		MachineType: runner.MachineTypeE1LargeAMD64,
		Steps: []runner.AgentStep{
			{Name: "Clone", Type: runner.AgentStepBash, Command: strPtr("git clone")},
		},
	}
	err := validateRunSuperPlaneSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt")
}

func TestValidateRunSuperPlaneSpecRejectsReservedEnv(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	value := "https://attacker.example"
	spec := RunSuperPlaneSpec{
		MachineType: runner.MachineTypeE1LargeAMD64,
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Environment: []runner.EnvironmentVariable{
			{Name: envAnthropicAPIKey, ValueSource: runner.EnvironmentValueSourceLiteral, Value: &value},
		},
	}
	err := validateRunSuperPlaneSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), envAnthropicAPIKey)
}

func cloneConfig(base map[string]any) map[string]any {
	out := make(map[string]any, len(base)+1)
	for key, value := range base {
		out[key] = value
	}
	return out
}

func strPtr(value string) *string {
	return &value
}
