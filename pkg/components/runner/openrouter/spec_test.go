package openrouter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestValidateRunOpenRouterSpecAcceptsIntegrationSource(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{
			Source:      runner.CredentialsSourceIntegration,
			Integration: configuration.IntegrationRef{Name: "openrouter"},
		},
		Model: "anthropic/claude-sonnet-4-6",
	}
	require.NoError(t, validateRunOpenRouterSpec(spec))
}

func TestValidateRunOpenRouterSpecAcceptsHostedCredentials(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
		Model:       "anthropic/claude-sonnet-4-6",
	}
	require.NoError(t, validateRunOpenRouterSpec(spec))
}

func TestValidateRunOpenRouterSpecRejectsHostedBaseURLEnv(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	value := "https://attacker.example"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
		Model:       "anthropic/claude-sonnet-4-6",
		Environment: []runner.EnvironmentVariable{
			{Name: envOpenRouterBaseURL, ValueSource: runner.EnvironmentValueSourceLiteral, Value: &value},
		},
	}
	err := validateRunOpenRouterSpec(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), envOpenRouterBaseURL)
}

func TestValidateRunOpenRouterSpecRequiresModel(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{
			Source:      runner.CredentialsSourceIntegration,
			Integration: configuration.IntegrationRef{Name: "openrouter"},
		},
	}
	err := validateRunOpenRouterSpec(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "model is required")
}

func TestDecodeRunOpenRouterSpecDefaultsMaxTurns(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunOpenRouterSpec(map[string]any{
		"machineType": "e1-large-amd64",
		"model":       "anthropic/claude-sonnet-4-6",
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": "fix tests"},
		},
		"credentials": map[string]any{"source": "hosted"},
	})
	require.NoError(t, err)
	require.Equal(t, DefaultMaxTurns, spec.MaxTurns)
	require.Equal(t, runner.DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
}

func TestDecodeRunOpenRouterSpecKeepsExplicitMaxTurns(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunOpenRouterSpec(map[string]any{
		"machineType": "e1-large-amd64",
		"model":       "anthropic/claude-sonnet-4-6",
		"maxTurns":    64,
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": "fix tests"},
		},
		"credentials": map[string]any{"source": "hosted"},
	})
	require.NoError(t, err)
	require.Equal(t, 64, spec.MaxTurns)
}

func TestValidateRunOpenRouterSpecRejectsMaxTurnsAboveLimit(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := validOpenRouterSpec(prompt)
	spec.MaxTurns = MaxTurnsLimit + 1
	err := validateRunOpenRouterSpec(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "max turns")
}

func TestValidateConfigurationOpenRouterRejectsMaxTurnsAboveLimit(t *testing.T) {
	t.Parallel()

	fields := (&RunOpenRouter{}).Configuration()
	err := configuration.ValidateConfiguration(fields, map[string]any{
		"machineType": "e1-large-amd64",
		"credentials": map[string]any{"source": "hosted"},
		"model":       "anthropic/claude-sonnet-4-6",
		"maxTurns":    MaxTurnsLimit + 1,
		"steps": []any{
			map[string]any{"name": "Prompt", "type": "prompt", "prompt": "fix tests"},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at most")
}

func TestBuildOpenRouterBrokerTaskPassesMaxTurns(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := validOpenRouterSpec(prompt)
	spec.MaxTurns = 64
	commands, _ := buildOpenRouterBrokerTask(spec)
	require.GreaterOrEqual(t, len(commands), 2)
	require.Contains(t, commands[1].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-prompt.txt" 'anthropic/claude-sonnet-4-6' 64`)
}

func TestBuildOpenRouterBrokerTaskDefaultsMaxTurns(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := validOpenRouterSpec(prompt)
	spec.MaxTurns = 0
	commands, _ := buildOpenRouterBrokerTask(spec)
	require.GreaterOrEqual(t, len(commands), 2)
	require.Contains(t, commands[1].Command, fmt.Sprintf("'anthropic/claude-sonnet-4-6' %d", DefaultMaxTurns))
}

func validOpenRouterSpec(prompt string) RunOpenRouterSpec {
	return RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
		Model:       "anthropic/claude-sonnet-4-6",
	}
}

func TestValidateConfigurationOpenRouterRequiresModel(t *testing.T) {
	t.Parallel()

	fields := (&RunOpenRouter{}).Configuration()
	err := configuration.ValidateConfiguration(fields, map[string]any{
		"machineType": "e1-large-amd64",
		"credentials": map[string]any{"source": "hosted"},
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": "fix tests"},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model")
}
