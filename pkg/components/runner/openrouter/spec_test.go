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

func TestValidateRunOpenRouterSpecRejectsHostedCredentials(t *testing.T) {
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
	err := validateRunOpenRouterSpec(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Run SuperPlane Agent")
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
		"credentials": map[string]any{
			"source":      "integration",
			"integration": map[string]any{"name": "openrouter"},
		},
		"model":    "anthropic/claude-sonnet-4-6",
		"maxTurns": MaxTurnsLimit + 1,
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
	task := buildOpenRouterBrokerTask(spec, "", nil)
	require.GreaterOrEqual(t, len(task.Commands), 2)
	require.Contains(t, task.Commands[1].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-prompt.txt" 'anthropic/claude-sonnet-4-6' 64`)
}

func TestBuildOpenRouterBrokerTaskDefaultsMaxTurns(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := validOpenRouterSpec(prompt)
	spec.MaxTurns = 0
	task := buildOpenRouterBrokerTask(spec, "", nil)
	require.GreaterOrEqual(t, len(task.Commands), 2)
	require.Contains(t, task.Commands[1].Command, fmt.Sprintf("'anthropic/claude-sonnet-4-6' %d", DefaultMaxTurns))
}

func TestApplyPlanningFollowUpLeavesLineAutomationsUnchanged(t *testing.T) {
	t.Parallel()

	spec := validOpenRouterSpec("fix tests")
	base := buildOpenRouterBrokerTask(spec, "", nil)
	got := applyPlanningFollowUp(base, nil, spec)
	require.Len(t, got.Commands, len(base.Commands))
	require.Len(t, got.Files, len(base.Files))
}

func TestApplyPlanningFollowUpAppendsWaitLoopForPlanningToken(t *testing.T) {
	t.Parallel()

	prompt := "greet"
	spec := validOpenRouterSpec(prompt)
	spec.MaxTurns = 32
	spec.Steps[0].WorkingDirectory = "repo"
	base := buildOpenRouterBrokerTask(spec, "", nil)
	got := applyPlanningFollowUp(base, []runner.BrokerEnvironmentVariable{{
		Name:  runner.EnvSuperplanePlanningID,
		Value: "session-1",
	}}, spec)

	require.Len(t, got.Commands, len(base.Commands)+1)
	last := got.Commands[len(got.Commands)-1]
	require.Equal(t, "Wait for the next message", last.Name)
	require.Equal(t, runner.LiveLogKindPrompt, last.Kind)
	require.Contains(t, last.Command, `node "$SUPERPLANE_TASK_DIR/follow_up_loop.js" 'anthropic/claude-sonnet-4-6' 32`)
	require.Contains(t, last.Command, `cd "$_sp_root"/'repo'`)

	var found bool
	for _, file := range got.Files {
		if file.Path == "follow_up_loop.js" {
			found = true
			require.Equal(t, runner.FollowUpLoopFile().Content, file.Content)
		}
	}
	require.True(t, found, "expected follow_up_loop.js task file")
}

func validOpenRouterSpec(prompt string) RunOpenRouterSpec {
	return RunOpenRouterSpec{
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
}

func TestValidateConfigurationOpenRouterRequiresModel(t *testing.T) {
	t.Parallel()

	fields := (&RunOpenRouter{}).Configuration()
	err := configuration.ValidateConfiguration(fields, map[string]any{
		"machineType": "e1-large-amd64",
		"credentials": map[string]any{
			"source":      "integration",
			"integration": map[string]any{"name": "openrouter"},
		},
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": "fix tests"},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model")
}
