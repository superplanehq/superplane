package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
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
	assert.Contains(t, req.Commands[0].Command, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`)
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
	assert.Equal(t, "sk-test-key", requireEnvironmentValue(t, req.Environment, envAnthropicAPIKey))
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

func TestRunClaudeCodeExecuteExtendsPromptsWithIntegrationUsageAndSetup(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-usage-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "sonnet",
			"steps": []map[string]any{
				{"name": "Fix tests", "type": "prompt", "prompt": "Fix the failing tests"},
			},
			"credentials": credentialsSecret("anthropic", "api_key"),
			"environmentFrom": []map[string]any{
				{"source": "integration", "integration": map[string]any{"name": "github-acme"}},
				{"source": "integration", "integration": map[string]any{"name": "semaphore-acme"}},
			},
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"anthropic/api_key": []byte("sk-test-key"),
			},
			IntegrationKeys: map[string]map[string][]byte{
				"github-acme":    {"GITHUB_TOKEN": []byte("gh-token")},
				"semaphore-acme": {"SEMAPHORE_API_TOKEN": []byte("sem-token")},
			},
			IntegrationUsage: map[string]string{
				"github-acme":    "The gh CLI is already installed. Use GITHUB_TOKEN.",
				"semaphore-acme": "Use sem-ai with SEMAPHORE_API_TOKEN.",
			},
			IntegrationSetup: map[string]string{
				"semaphore-acme": "echo install-sem-ai",
			},
			IntegrationSetupName: map[string]string{
				"semaphore-acme": "Set up Semaphore",
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

	require.Len(t, req.Commands, 3)
	assert.Equal(t, "Prepare Claude Code", req.Commands[0].Name)
	assert.Equal(t, "Set up Semaphore", req.Commands[1].Name)
	assert.Equal(t, runner.LiveLogKindSetup, req.Commands[1].Kind)
	assert.Equal(t, "Set up Semaphore", req.Commands[1].Preview)
	assert.Equal(t, "Fix tests", req.Commands[2].Name)
	assert.Equal(t, "Fix the failing tests", req.Commands[2].Preview)
	assert.Equal(t, "gh-token", requireEnvironmentValue(t, req.Environment, "GITHUB_TOKEN"))
	assert.Equal(t, "sem-token", requireEnvironmentValue(t, req.Environment, "SEMAPHORE_API_TOKEN"))
	assert.Equal(t, "echo install-sem-ai", requireTaskFile(t, req.Files, "setup/01-set-up-semaphore.sh").Content)
	assert.Equal(
		t,
		"The gh CLI is already installed. Use GITHUB_TOKEN.\n\nUse sem-ai with SEMAPHORE_API_TOKEN.\n\nFix the failing tests",
		requireTaskFile(t, req.Files, "prompts/01-fix-tests.txt").Content,
	)
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
	assert.Contains(t, req.Commands[0].Command, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`)
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

// A BYOK run spends the key of the organization, not SuperPlane credit, so no
// selected-model list stands between it and the provider. An organization that
// connects a key must be able to run without an administrator selecting models
// first, and an agent CLI alias such as "opus" must reach the provider as it is.
func TestRunClaudeCodeExecuteDoesNotGateBYOKModels(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-claude-byok-1"}`))},
		},
	}

	component := &RunClaudeCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "opus",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
			"credentials": credentialsSecret("anthropic", "api_key"),
		},
		HTTP:    httpContext,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{"anthropic/api_key": []byte("sk-byok")}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			SelectableErr: fmt.Errorf("model opus is not on the selected-model list"),
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
	assert.Equal(t, "sk-byok", requireEnvironmentValue(t, req.Environment, envAnthropicAPIKey))
}

func TestRunClaudeCodeExecuteRejectsHostedCredentials(t *testing.T) {
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
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		HostedLLM:      &contexts.HostedLLMContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Run SuperPlane Agent")
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
