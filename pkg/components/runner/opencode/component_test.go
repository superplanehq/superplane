package opencode

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
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testRunnerMachineType = runner.MachineTypeE1LargeAMD64

func secretConfig(secret, key string) map[string]any {
	return map[string]any{
		"secret": secret,
		"key":    key,
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

func TestRunOpenCodeExecuteSendsPerStepCommandsToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-opencode-1"}`))},
		},
	}

	component := &RunOpenCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"provider":    "openai",
			"secret":      secretConfig("openai", "api_key"),
			"model":       "gpt-4.1",
			"steps": []map[string]any{
				{"name": "Clone", "type": "bash", "command": "git clone https://github.com/acme/widgets.git /tmp/repo"},
				{"name": "Fix tests", "type": "prompt", "prompt": "Fix the failing tests"},
				{"name": "Open PR", "type": "prompt", "prompt": "Open a pull request"},
				{"name": "Status", "type": "bash", "command": "git -C /tmp/repo status"},
			},
			"workingDirectory": "/tmp",
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"openai/api_key": []byte("sk-test-key"),
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
	assert.Equal(t, "Prepare OpenCode", req.Commands[0].Name)
	assert.Equal(t, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`, req.Commands[0].Command)
	assert.Equal(t, runner.BrokerCommand{Name: "Clone", Command: `source "$SUPERPLANE_TASK_DIR/steps/01-clone.sh"`}, req.Commands[1])
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Fix tests",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-tests.txt" 'openai/gpt-4.1'`,
	}, req.Commands[2])
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Open PR",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/03-open-pr.txt" 'openai/gpt-4.1'`,
	}, req.Commands[3])
	assert.Equal(t, runner.BrokerCommand{Name: "Status", Command: `source "$SUPERPLANE_TASK_DIR/steps/04-status.sh"`}, req.Commands[4])
	assert.Contains(t, string(body), `"name":"Clone"`)
	assert.Empty(t, req.DockerImage)
	require.Len(t, req.Environment, 1)
	assert.Equal(t, "OPENAI_API_KEY", req.Environment[0].Name)
	assert.Equal(t, "sk-test-key", req.Environment[0].Value)
	assert.NotContains(t, string(body), `"message_chain"`)

	require.Len(t, req.Files, 6)
	assert.Equal(t, runScript, requireTaskFile(t, req.Files, "run.js").Content)
	assert.Contains(t, requireTaskFile(t, req.Files, "prepare.sh").Content, "cd '/tmp'")
	assert.Equal(t, "git clone https://github.com/acme/widgets.git /tmp/repo", requireTaskFile(t, req.Files, "steps/01-clone.sh").Content)
	assert.Equal(t, "Fix the failing tests", requireTaskFile(t, req.Files, "prompts/02-fix-tests.txt").Content)
	assert.Equal(t, "Open a pull request", requireTaskFile(t, req.Files, "prompts/03-open-pr.txt").Content)
	assert.Equal(t, "git -C /tmp/repo status", requireTaskFile(t, req.Files, "steps/04-status.sh").Content)
}

func TestRunOpenCodeExecuteInjectsProviderKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-opencode-2"}`))},
		},
	}

	component := &RunOpenCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"provider":    "anthropic",
			"secret":      secretConfig("anthropic", "api_key"),
			"model":       "claude-sonnet-4-5",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"anthropic/api_key": []byte("sk-ant"),
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

	require.Len(t, req.Environment, 1)
	assert.Equal(t, "ANTHROPIC_API_KEY", req.Environment[0].Name)
	assert.Equal(t, "sk-ant", req.Environment[0].Value)

	// The bare model name is composed with the provider into provider/model.
	require.Len(t, req.Commands, 2)
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Hello",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-hello.txt" 'anthropic/claude-sonnet-4-5'`,
	}, req.Commands[1])
}

func TestRunOpenCodeExecuteInjectsCloudflareGatewayEnv(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-opencode-cf"}`))},
		},
	}

	component := &RunOpenCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType":         testRunnerMachineType,
			"provider":            providerCloudflareAIGateway,
			"secret":              secretConfig("cloudflare", "api_token"),
			"cloudflareAccountId": "acct-123",
			"cloudflareGatewayId": "my-gateway",
			"model":               "moonshotai/kimi-k3",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP: httpContext,
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"cloudflare/api_token": []byte("cf-token"),
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

	env := map[string]string{}
	for _, variable := range req.Environment {
		env[variable.Name] = variable.Value
	}
	assert.Equal(t, "cf-token", env[envCloudflareAPIToken])
	assert.Equal(t, "acct-123", env[envCloudflareAccountID])
	assert.Equal(t, "my-gateway", env[envCloudflareGatewayID])

	require.Len(t, req.Commands, 2)
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Hello",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-hello.txt" 'cloudflare-ai-gateway/moonshotai/kimi-k3'`,
	}, req.Commands[1])

	config := requireTaskFile(t, req.Files, "opencode.jsonc").Content
	assert.Contains(t, config, `"moonshotai/kimi-k3"`)
	assert.NotContains(t, config, "cf-token")
	for _, file := range req.Files {
		assert.NotContains(t, file.Content, "cf-token")
	}
}

func TestRunOpenCodeExecuteRequiresProviderSecret(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	component := &RunOpenCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"provider":    "openai",
			"secret":      secretConfig("openai", "api_key"),
			"model":       "gpt-4.1",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
}

func TestRunOpenCodeExecuteRequiresModel(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	component := &RunOpenCode{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"provider":    "openai",
			"secret":      secretConfig("openai", "api_key"),
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"openai/api_key": []byte("k")}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestRunOpenCodeProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &runner.Task{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"result":"done","session_id":"ses_abc"}`),
	}
	require.NoError(t, runner.ProcessBrokerTaskStatus(state, task, FinishedEventType))
	require.Equal(t, runner.PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	assert.Equal(t, FinishedEventType, wrapped["type"])
}
