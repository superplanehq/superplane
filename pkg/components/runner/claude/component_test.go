package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testRunnerMachineType = runner.MachineTypeE1LargeAMD64

func credentialsSecret(secret, key string) map[string]any {
	return map[string]any{
		"source": "secret",
		"secret": map[string]any{
			"secret": secret,
			"key":    key,
		},
	}
}

type createTaskRequest struct {
	FleetID       string                             `json:"fleet_id"`
	RunMode       string                             `json:"run_mode,omitempty"`
	Script        string                             `json:"script,omitempty"`
	MessageChain  json.RawMessage                    `json:"message_chain,omitempty"`
	Commands      []runner.BrokerCommand             `json:"commands,omitempty"`
	SetupCommands []string                           `json:"setup_commands,omitempty"`
	Environment   []runner.BrokerEnvironmentVariable `json:"environment,omitempty"`
	Files         []runner.BrokerTaskFile            `json:"files,omitempty"`
	ExecutionMode string                             `json:"execution_mode,omitempty"`
	DockerImage   string                             `json:"docker_image,omitempty"`
}

func TestRunClaudeCodeExecuteSendsPerStepCommandsToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "sonnet",
			"steps": []map[string]any{
				{"name": "Clone", "type": "bash", "command": "git clone https://github.com/acme/widgets.git /tmp/repo"},
				{"name": "Fix tests", "type": "prompt", "prompt": "Fix the failing tests"},
				{"name": "Open PR", "type": "prompt", "prompt": "Open a pull request"},
				{"name": "Status", "type": "bash", "command": "git -C /tmp/repo status"},
			},
			"credentials":      credentialsSecret("anthropic", "api_key"),
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

	var req createTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, testRunnerMachineType, req.FleetID)
	assert.Empty(t, req.RunMode)
	assert.Empty(t, req.Script)
	assert.Empty(t, req.SetupCommands)
	assert.Empty(t, req.MessageChain)
	assert.Equal(t, runner.ExecutionModeHost, req.ExecutionMode)
	require.Len(t, req.Commands, 5)
	assert.Equal(t, "Prepare Claude Code", req.Commands[0].Name)
	assert.Equal(t, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`, req.Commands[0].Command)
	assert.Equal(t, "Clone", req.Commands[1].Name)
	assert.Contains(t, req.Commands[1].Command, `source "$SUPERPLANE_TASK_DIR/steps/01-clone.sh"`)
	assert.Contains(t, req.Commands[1].Command, `node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge`)
	assert.Equal(t, "Fix tests", req.Commands[2].Name)
	assert.Contains(t, req.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-tests.txt" 'sonnet'`)
	assert.Equal(t, "Open PR", req.Commands[3].Name)
	assert.Contains(t, req.Commands[3].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/03-open-pr.txt" 'sonnet'`)
	assert.Equal(t, "Status", req.Commands[4].Name)
	assert.Contains(t, req.Commands[4].Command, `source "$SUPERPLANE_TASK_DIR/steps/04-status.sh"`)
	assert.Contains(t, string(body), `"name":"Clone"`)
	assert.Empty(t, req.DockerImage)
	require.Len(t, req.Environment, 1)
	assert.Equal(t, envAnthropicAPIKey, req.Environment[0].Name)
	assert.Equal(t, "sk-test-key", req.Environment[0].Value)
	assert.NotContains(t, string(body), `"message_chain"`)

	require.Len(t, req.Files, 7)
	assert.Equal(t, runScript, requireTaskFile(t, req.Files, "run.js").Content)
	assert.Equal(t, runner.LLMUsageScript, requireTaskFile(t, req.Files, "llm_usage.js").Content)
	assert.Contains(t, requireTaskFile(t, req.Files, "prepare.sh").Content, "cd '/tmp'")
	assert.Contains(t, requireTaskFile(t, req.Files, "prepare.sh").Content, `pwd -P >"$SUPERPLANE_TASK_DIR/task_cwd"`)
	assert.Equal(t, "git clone https://github.com/acme/widgets.git /tmp/repo", requireTaskFile(t, req.Files, "steps/01-clone.sh").Content)
	assert.Equal(t, "Fix the failing tests", requireTaskFile(t, req.Files, "prompts/02-fix-tests.txt").Content)
	assert.Equal(t, "Open a pull request", requireTaskFile(t, req.Files, "prompts/03-open-pr.txt").Content)
	assert.Equal(t, "git -C /tmp/repo status", requireTaskFile(t, req.Files, "steps/04-status.sh").Content)
}

func TestRunClaudeCodeExecuteMigratesLegacyPromptConfig(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-legacy-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType":           testRunnerMachineType,
			"prompt":                "implement the issue",
			"enable_setup_commands": true,
			"setup_commands":        "git clone https://github.com/acme/widgets.git /tmp/repo",
			"enable_after_commands": true,
			"after_commands":        "git push",
			"credentials":           credentialsSecret("anthropic", "api_key"),
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

	var req createTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	assert.Empty(t, req.SetupCommands)
	assert.Empty(t, req.MessageChain)
	require.Len(t, req.Commands, 4)
	assert.Equal(t, "Prepare Claude Code", req.Commands[0].Name)
	assert.Equal(t, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`, req.Commands[0].Command)
	assert.Equal(t, "Setup", req.Commands[1].Name)
	assert.Contains(t, req.Commands[1].Command, `source "$SUPERPLANE_TASK_DIR/steps/01-setup.sh"`)
	assert.Equal(t, "Prompt", req.Commands[2].Name)
	assert.Contains(t, req.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-prompt.txt" ''`)
	assert.Equal(t, "After", req.Commands[3].Name)
	assert.Contains(t, req.Commands[3].Command, `source "$SUPERPLANE_TASK_DIR/steps/03-after.sh"`)
	require.Len(t, req.Files, 6)
	assert.Equal(t, "git clone https://github.com/acme/widgets.git /tmp/repo", requireTaskFile(t, req.Files, "steps/01-setup.sh").Content)
	assert.Equal(t, "implement the issue", requireTaskFile(t, req.Files, "prompts/02-prompt.txt").Content)
	assert.Equal(t, "git push", requireTaskFile(t, req.Files, "steps/03-after.sh").Content)
}

func TestRunClaudeCodeExecuteRequiresAPIKeySecret(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
			"credentials": credentialsSecret("anthropic", "api_key"),
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestRunClaudeCodeExecuteInjectsHostedAPIKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-hosted-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "claude-sonnet-4-6",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
			"credentials": map[string]any{"source": "hosted"},
		},
		HTTP:    httpContext,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{
				APIKey:        "sk-hosted",
				AllowedModels: []string{"claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	var req createTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	require.Len(t, req.Environment, 1)
	assert.Equal(t, envAnthropicAPIKey, req.Environment[0].Name)
	assert.Equal(t, "sk-hosted", req.Environment[0].Value)
}

func TestRunClaudeCodeExecuteSoftBlocksWhenHostedCreditIsEmpty(t *testing.T) {
	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "claude-sonnet-4-6",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
			"credentials": map[string]any{"source": "hosted"},
		},
		HTTP:    &contexts.HTTPContext{},
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			CreditErr: models.ErrHostedCreditEmpty,
			Access: core.HostedLLMAccess{
				APIKey:        "sk-hosted",
				AllowedModels: []string{"claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrHostedCreditEmpty)
}

func TestRunClaudeCodeProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &runner.Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"result":"done","session_id":"abc"}`),
	}
	require.NoError(t, runner.ProcessBrokerTaskStatus(state, task, FinishedEventType, ""))
	require.Equal(t, runner.PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, FinishedEventType, wrapped["type"])
}
