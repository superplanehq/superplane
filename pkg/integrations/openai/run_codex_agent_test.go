package openai

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/agentcli"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type fakeCodexRunner struct {
	command agentcli.Command
	result  agentcli.Result
	err     error
}

func (r *fakeCodexRunner) Run(ctx context.Context, command agentcli.Command) (agentcli.Result, error) {
	r.command = command
	return r.result, r.err
}

func TestRunCodexAgent_Setup(t *testing.T) {
	workingDirectory := t.TempDir()

	tests := []struct {
		name          string
		configuration map[string]any
		expectedError string
	}{
		{
			name: "valid",
			configuration: map[string]any{
				"prompt":           "Review the repository",
				"workingDirectory": workingDirectory,
			},
		},
		{
			name: "missing prompt",
			configuration: map[string]any{
				"workingDirectory": workingDirectory,
			},
			expectedError: "prompt is required",
		},
		{
			name: "invalid sandbox",
			configuration: map[string]any{
				"prompt":           "Review",
				"sandbox":          "invalid",
				"workingDirectory": workingDirectory,
			},
			expectedError: "sandbox must be one of",
		},
		{
			name: "invalid timeout",
			configuration: map[string]any{
				"prompt":           "Review",
				"workingDirectory": workingDirectory,
				"timeoutSeconds":   maxCodexTimeoutSeconds + 1,
			},
			expectedError: "timeoutSeconds must be between",
		},
		{
			name: "missing working directory",
			configuration: map[string]any{
				"prompt":           "Review",
				"workingDirectory": workingDirectory + "/missing",
			},
			expectedError: "workingDirectory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := &RunCodexAgent{}
			err := component.Setup(core.SetupContext{Configuration: tt.configuration})
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRunCodexAgent_ExecuteSuccess(t *testing.T) {
	workingDirectory := t.TempDir()
	runner := &fakeCodexRunner{
		result: agentcli.Result{
			Stdout:   `{"type":"agent_message","message":"Finished review."}` + "\n",
			ExitCode: 0,
			Duration: 1500 * time.Millisecond,
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	component := &RunCodexAgent{runner: runner}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"model":            "gpt-5.1-codex-mini",
			"prompt":           "Review the repository",
			"sandbox":          "read-only",
			"workingDirectory": workingDirectory,
			"timeoutSeconds":   30,
		},
		ExecutionState: execState,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-test"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "codex", runner.command.Name)
	assert.Equal(t, workingDirectory, runner.command.Dir)
	assert.Equal(t, 30*time.Second, runner.command.Timeout)
	assert.Equal(t, "sk-test", runner.command.Env["CODEX_API_KEY"])
	assert.Equal(t, "sk-test", runner.command.Env["OPENAI_API_KEY"])
	assert.Contains(t, runner.command.Args, "--json")
	assert.Contains(t, runner.command.Args, "--ephemeral")
	assert.Contains(t, runner.command.Args, "--output-last-message")
	assert.Contains(t, runner.command.Args, "Review the repository")

	assert.Equal(t, CodexAgentOutputChannelSuccess, execState.Channel)
	assert.Equal(t, CodexAgentPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	wrapped := execState.Payloads[0].(map[string]any)
	payload := wrapped["data"].(CodexAgentPayload)
	assert.Equal(t, "Finished review.", payload.Text)
	assert.Equal(t, 0, payload.ExitCode)
	assert.False(t, payload.TimedOut)
	assert.Equal(t, int64(1500), payload.DurationMs)
	assert.Empty(t, payload.Stderr)
	require.Len(t, payload.Events, 1)
	assert.Equal(t, "agent_message", payload.Events[0]["type"])
}

func TestRunCodexAgent_ExecuteRoutesNonZeroExitToFailed(t *testing.T) {
	workingDirectory := t.TempDir()
	runner := &fakeCodexRunner{
		result: agentcli.Result{
			Stdout:   "not json",
			Stderr:   "Codex failed",
			ExitCode: 2,
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	component := &RunCodexAgent{runner: runner}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"prompt":           "Review",
			"workingDirectory": workingDirectory,
		},
		ExecutionState: execState,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-test"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, CodexAgentOutputChannelFailed, execState.Channel)
	wrapped := execState.Payloads[0].(map[string]any)
	payload := wrapped["data"].(CodexAgentPayload)
	assert.Equal(t, 2, payload.ExitCode)
	assert.Equal(t, "not json", payload.Stdout)
	assert.Equal(t, "Codex failed", payload.Stderr)
}

func TestRunCodexAgent_ExecuteRunnerError(t *testing.T) {
	workingDirectory := t.TempDir()
	component := &RunCodexAgent{runner: &fakeCodexRunner{err: errors.New("not found")}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"prompt":           "Review",
			"workingDirectory": workingDirectory,
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-test"},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run Codex CLI")
}

func TestOpenAI_ActionsIncludesRunCodexAgent(t *testing.T) {
	actions := (&OpenAI{}).Actions()
	found := slices.ContainsFunc(actions, func(action core.Action) bool {
		return action.Name() == "openai.runCodexAgent"
	})
	assert.True(t, found)
}
