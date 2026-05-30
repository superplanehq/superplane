package claude

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

type fakeClaudeCodeRunner struct {
	command agentcli.Command
	result  agentcli.Result
	err     error
}

func (r *fakeClaudeCodeRunner) Run(ctx context.Context, command agentcli.Command) (agentcli.Result, error) {
	r.command = command
	return r.result, r.err
}

func TestRunClaudeCode_Setup(t *testing.T) {
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
			name: "invalid permission mode",
			configuration: map[string]any{
				"prompt":           "Review",
				"permissionMode":   "invalid",
				"workingDirectory": workingDirectory,
			},
			expectedError: "permissionMode must be one of",
		},
		{
			name: "invalid max turns",
			configuration: map[string]any{
				"prompt":           "Review",
				"workingDirectory": workingDirectory,
				"maxTurns":         maxClaudeCodeTurns + 1,
			},
			expectedError: "maxTurns must be between",
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
			component := &RunClaudeCode{}
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

func TestRunClaudeCode_ExecuteSuccess(t *testing.T) {
	workingDirectory := t.TempDir()
	runner := &fakeClaudeCodeRunner{
		result: agentcli.Result{
			Stdout:   `{"type":"result","subtype":"success","is_error":false,"result":"Finished implementation."}`,
			ExitCode: 0,
			Duration: 2 * time.Second,
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	component := &RunClaudeCode{runner: runner}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"model":            "sonnet",
			"prompt":           "Implement the feature",
			"permissionMode":   "plan",
			"workingDirectory": workingDirectory,
			"timeoutSeconds":   45,
			"maxTurns":         4,
		},
		ExecutionState: execState,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-ant-test"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "claude", runner.command.Name)
	assert.Equal(t, workingDirectory, runner.command.Dir)
	assert.Equal(t, 45*time.Second, runner.command.Timeout)
	assert.Equal(t, "sk-ant-test", runner.command.Env["ANTHROPIC_API_KEY"])
	assert.Equal(t, "1", runner.command.Env["DISABLE_AUTOUPDATER"])
	assert.Contains(t, runner.command.Args, "--bare")
	assert.Contains(t, runner.command.Args, "--no-session-persistence")
	assert.Contains(t, runner.command.Args, "--permission-mode")
	assert.Contains(t, runner.command.Args, "plan")
	assert.Contains(t, runner.command.Args, "--max-turns")
	assert.Contains(t, runner.command.Args, "4")
	assert.Contains(t, runner.command.Args, "Implement the feature")

	assert.Equal(t, ClaudeCodeOutputChannelSuccess, execState.Channel)
	assert.Equal(t, ClaudeCodePayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	wrapped := execState.Payloads[0].(map[string]any)
	payload := wrapped["data"].(ClaudeCodePayload)
	assert.Equal(t, "Finished implementation.", payload.Text)
	assert.Equal(t, 0, payload.ExitCode)
	assert.False(t, payload.TimedOut)
	assert.False(t, payload.IsError)
	assert.Equal(t, int64(2000), payload.DurationMs)
	assert.Equal(t, "success", payload.Response["subtype"])
}

func TestRunClaudeCode_ExecuteRoutesErrorResponseToFailed(t *testing.T) {
	workingDirectory := t.TempDir()
	runner := &fakeClaudeCodeRunner{
		result: agentcli.Result{
			Stdout:   `{"type":"result","subtype":"error_max_turns","is_error":true,"result":"Reached max turns"}`,
			ExitCode: 0,
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	component := &RunClaudeCode{runner: runner}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"prompt":           "Review",
			"workingDirectory": workingDirectory,
		},
		ExecutionState: execState,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-ant-test"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, ClaudeCodeOutputChannelFailed, execState.Channel)
	wrapped := execState.Payloads[0].(map[string]any)
	payload := wrapped["data"].(ClaudeCodePayload)
	assert.True(t, payload.IsError)
	assert.Equal(t, "Reached max turns", payload.Text)
}

func TestRunClaudeCode_ExecuteRunnerError(t *testing.T) {
	workingDirectory := t.TempDir()
	component := &RunClaudeCode{runner: &fakeClaudeCodeRunner{err: errors.New("not found")}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"prompt":           "Review",
			"workingDirectory": workingDirectory,
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-ant-test"},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run Claude Code CLI")
}

func TestClaude_ActionsIncludesRunClaudeCode(t *testing.T) {
	actions := (&Claude{}).Actions()
	found := slices.ContainsFunc(actions, func(action core.Action) bool {
		return action.Name() == "claude.runClaudeCode"
	})
	assert.True(t, found)
}
