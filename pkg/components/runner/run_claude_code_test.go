package runner

import (
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

func TestRunClaudeCodeExecuteSendsBashTaskToBroker(t *testing.T) {
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
		Expressions: &stubMessageChainBuilder{
			chain: map[string]any{
				"Issue": map[string]any{"data": map[string]any{"title": "bug"}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, testRunnerMachineType, req.FleetID)
	assert.Equal(t, RunModeBash, req.RunMode)
	assert.Equal(t, ExecutionModeHost, req.ExecutionMode)
	assert.Empty(t, req.SetupCommands)
	assert.Contains(t, req.Script, "claude")
	assert.Contains(t, req.Script, "--output-format stream-json")
	assert.Contains(t, req.Script, "--verbose")
	assert.Contains(t, req.Script, "--include-partial-messages")
	assert.Contains(t, req.Script, "cd '/tmp'")
	assert.Contains(t, req.Script, "git clone https://github.com/acme/widgets.git /tmp/repo")
	assert.Contains(t, req.Script, "--continue")
	assert.Contains(t, req.Script, "git -C /tmp/repo status")
	assert.Empty(t, req.DockerImage)
	require.Len(t, req.Environment, 1)
	assert.Equal(t, envAnthropicAPIKey, req.Environment[0].Name)
	assert.Equal(t, "sk-test-key", req.Environment[0].Value)
	require.True(t, json.Valid(req.MessageChain))
	assert.Contains(t, string(req.MessageChain), "Issue")
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
		Expressions:    &stubMessageChainBuilder{chain: map[string]any{}},
	})
	require.NoError(t, err)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	assert.Empty(t, req.SetupCommands)
	assert.Contains(t, req.Script, "git clone https://github.com/acme/widgets.git /tmp/repo")
	assert.Contains(t, req.Script, "claude")
	assert.Contains(t, req.Script, "git push")
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
