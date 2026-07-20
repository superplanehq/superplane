package runner

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRunClaudeCodeExecuteSendsPerStepCommandsToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"model":        "sonnet",
			"steps": []map[string]any{
				{"name": "Clone", "type": "bash", "command": "git clone https://github.com/acme/widgets.git /tmp/repo"},
				{"name": "Fix tests", "type": "prompt", "prompt": "Fix the failing tests"},
				{"name": "Open PR", "type": "prompt", "prompt": "Open a pull request"},
				{"name": "Status", "type": "bash", "command": "git -C /tmp/repo status"},
			},
			"anthropicApiKey": map[string]any{
				"secret": "anthropic",
				"key":    "api_key",
			},
			"workingDirectory": "/tmp",
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"anthropic/api_key": []byte("sk-test-key"),
			},
		},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, testRunnerMachineType, req.FleetID)
	assert.Empty(t, req.RunMode)
	assert.Empty(t, req.Script)
	assert.Empty(t, req.SetupCommands)
	assert.Empty(t, req.MessageChain)
	assert.Equal(t, ExecutionModeHost, req.ExecutionMode)
	require.Len(t, req.Commands, 5)
	assert.Equal(t, "Prepare Claude Code", req.Commands[0].Name)
	assert.True(t, strings.HasPrefix(req.Commands[0].Command, "bash -c "))
	assert.Contains(t, req.Commands[0].Command, "claude CLI not found")
	assert.Contains(t, req.Commands[0].Command, "/tmp")
	assert.Equal(t, BrokerCommand{Name: "Clone", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/01-clone.sh"`}, req.Commands[1])
	assert.Equal(t, BrokerCommand{Name: "Fix tests", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/02-fix-tests.sh"`}, req.Commands[2])
	assert.Equal(t, BrokerCommand{Name: "Open PR", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/03-open-pr.sh"`}, req.Commands[3])
	assert.Equal(t, BrokerCommand{Name: "Status", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/04-status.sh"`}, req.Commands[4])
	assert.Contains(t, req.Commands[0].Command, base64OfClaudeBashStep("git clone https://github.com/acme/widgets.git /tmp/repo"))
	assert.Contains(t, req.Commands[0].Command, base64OfClaudePromptStep("Fix the failing tests", "sonnet"))
	assert.Contains(t, req.Commands[0].Command, base64OfClaudeBashStep("git -C /tmp/repo status"))
	assert.Contains(t, string(body), `"name":"Clone"`)
	assert.Empty(t, req.DockerImage)
	require.Len(t, req.Environment, 1)
	assert.Equal(t, envAnthropicAPIKey, req.Environment[0].Name)
	assert.Equal(t, "sk-test-key", req.Environment[0].Value)
	assert.NotContains(t, string(body), `"message_chain"`)
}

func TestRunClaudeCodeExecuteMigratesLegacyPromptConfig(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-legacy-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type":          testRunnerMachineType,
			"prompt":                "implement the issue",
			"enable_setup_commands": true,
			"setup_commands":        "git clone https://github.com/acme/widgets.git /tmp/repo",
			"enable_after_commands": true,
			"after_commands":        "git push",
			"anthropicApiKey": map[string]any{
				"secret": "anthropic",
				"key":    "api_key",
			},
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"anthropic/api_key": []byte("sk-test-key"),
			},
		},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	assert.Empty(t, req.SetupCommands)
	assert.Empty(t, req.MessageChain)
	require.Len(t, req.Commands, 4)
	assert.Equal(t, "Prepare Claude Code", req.Commands[0].Name)
	assert.True(t, strings.HasPrefix(req.Commands[0].Command, "bash -c "))
	assert.Equal(t, BrokerCommand{Name: "Setup", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/01-setup.sh"`}, req.Commands[1])
	assert.Equal(t, BrokerCommand{Name: "Prompt", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/02-prompt.sh"`}, req.Commands[2])
	assert.Equal(t, BrokerCommand{Name: "After", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/03-after.sh"`}, req.Commands[3])
	assert.Contains(t, req.Commands[0].Command, base64OfClaudeBashStep("git clone https://github.com/acme/widgets.git /tmp/repo"))
	assert.Contains(t, req.Commands[0].Command, base64OfClaudePromptStep("implement the issue", ""))
	assert.Contains(t, req.Commands[0].Command, base64OfClaudeBashStep("git push"))
}

func base64OfClaudeBashStep(command string) string {
	return base64.StdEncoding.EncodeToString([]byte(buildClaudeBashStepScript(command)))
}

func base64OfClaudePromptStep(prompt, model string) string {
	return base64.StdEncoding.EncodeToString([]byte(buildClaudePromptStepScript(prompt, model)))
}

func TestRunClaudeCodeExecuteRequiresAPIKeySecret(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
			"anthropicApiKey": map[string]any{
				"secret": "anthropic",
				"key":    "api_key",
			},
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
		Expressions:    &stubMessageChainBuilder{chain: map[string]any{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "anthropic API key")
}

func TestRunClaudeCodeProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"result":"done","session_id":"abc"}`),
	}
	require.NoError(t, processBrokerTaskStatus(state, task, RunClaudeCodeFinishedEventType))
	require.Equal(t, PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, RunClaudeCodeFinishedEventType, wrapped["type"])
}
