package superplane

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

type createTaskRequest struct {
	Commands    []runner.BrokerCommand             `json:"commands,omitempty"`
	Environment []runner.BrokerEnvironmentVariable `json:"environment,omitempty"`
	Files       []runner.BrokerTaskFile            `json:"files,omitempty"`
}

func TestRunSuperPlaneExecuteDispatchesClaude(t *testing.T) {
	req := executeSuperPlane(t, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderAnthropic,
		Model:    "claude-sonnet-4-6",
	}, core.HostedLLMAccess{
		APIKey:        "sk-hosted",
		AllowedModels: []string{"claude-sonnet-4-6"},
	})
	assert.Equal(t, "sk-hosted", requireEnvironmentValue(t, req.Environment, envAnthropicAPIKey))
	assert.True(t, hasTaskFile(req.Files, "run.js"))
}

func TestRunSuperPlaneExecuteDispatchesCodex(t *testing.T) {
	req := executeSuperPlane(t, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderOpenAI,
		Model:    "gpt-5",
	}, core.HostedLLMAccess{
		APIKey:        "sk-openai",
		AllowedModels: []string{"gpt-5"},
	})
	assert.Equal(t, "sk-openai", requireEnvironmentValue(t, req.Environment, envOpenAIAPIKey))
}

func TestRunSuperPlaneExecuteDispatchesOpenRouter(t *testing.T) {
	req := executeSuperPlane(t, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderOpenRouter,
		Model:    "anthropic/claude-sonnet-4-6",
	}, core.HostedLLMAccess{
		APIKey:        "sk-or",
		AllowedModels: []string{"anthropic/claude-sonnet-4-6"},
	})
	assert.Equal(t, "sk-or", requireEnvironmentValue(t, req.Environment, envOpenRouterAPIKey))
}

func TestRunSuperPlaneExecuteRejectsMissingDefaultModel(t *testing.T) {
	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		HostedLLM:      &contexts.HostedLLMContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrSuperPlaneRunnerNoModel)
}

func TestRunSuperPlaneExecuteSoftBlocksWhenHostedCreditIsEmpty(t *testing.T) {
	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:    &contexts.HTTPContext{},
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Default: core.DefaultHostedLLMModel{
				Provider: models.UsageProviderAnthropic,
				Model:    "claude-sonnet-4-6",
			},
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

func executeSuperPlane(t *testing.T, defaultModel core.DefaultHostedLLMModel, access core.HostedLLMAccess) createTaskRequest {
	t.Helper()
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-superplane-1"}`))},
		},
	}

	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:    httpContext,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Default: defaultModel,
			Access:  access,
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
	return req
}

func requireEnvironmentValue(t *testing.T, environment []runner.BrokerEnvironmentVariable, name string) string {
	t.Helper()
	for _, variable := range environment {
		if variable.Name == name {
			return variable.Value
		}
	}
	t.Fatalf("missing environment %s", name)
	return ""
}

func hasTaskFile(files []runner.BrokerTaskFile, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func TestRunSuperPlaneExecuteRejectsPrivateHostedBaseURL(t *testing.T) {
	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:    &contexts.HTTPContext{},
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Default: core.DefaultHostedLLMModel{
				Provider: models.UsageProviderAnthropic,
				Model:    "claude-sonnet-4-6",
			},
			Access: core.HostedLLMAccess{
				APIKey:        "sk-hosted",
				BaseURL:       "http://127.0.0.1/v1",
				AllowedModels: []string{"claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}
