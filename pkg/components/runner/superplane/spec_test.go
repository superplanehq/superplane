package superplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

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

	for _, key := range []string{"credentials", "model", "maxTurns", "hostedProvider"} {
		cfg := cloneConfig(base)
		cfg[key] = "not-allowed"
		_, err := decodeRunSuperPlaneSpec(cfg)
		require.Error(t, err, key)
		assert.Contains(t, err.Error(), key)
	}
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
