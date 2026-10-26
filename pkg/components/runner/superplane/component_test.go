package superplane

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	openrouterapi "github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testRunnerMachineType = runner.MachineTypeE1LargeAMD64

type createTaskRequest struct {
	Commands    []runner.TaskCommand             `json:"commands,omitempty"`
	Environment []runner.TaskEnvironmentVariable `json:"environment,omitempty"`
	Files       []runner.TaskFile                `json:"files,omitempty"`
}

func TestRunSuperPlaneDispatchesAdditionalInstructions(t *testing.T) {
	planningPrompt := "Review and score this task\n\nTask details"
	for _, instructions := range []string{"Avoid new dependencies. Document {{ example }} in the PR.", "", " \n "} {
		t.Run(instructions, func(t *testing.T) {
			configuration := map[string]any{
				"machineType":            testRunnerMachineType,
				"additionalInstructions": instructions,
				"environment": []map[string]any{{
					"name": runner.EnvSuperplanePlanningID, "value": "session-1", "valueSource": "literal",
				}},
				"steps": []map[string]any{
					{"name": "Implementation", "type": "prompt", "prompt": "Implement the task"},
					{"name": "Push", "type": "bash", "command": "git push"},
					{"name": "Write PR", "type": "prompt", "prompt": "Write the PR"},
					{"name": "Plan Task", "type": "prompt", "prompt": planningPrompt},
				},
			}
			resolved, err := workercontexts.NewNodeConfigurationBuilder(nil, uuid.Nil).
				WithConfigurationFields((&RunSuperPlane{}).Configuration()).Build(configuration)
			require.NoError(t, err)
			req := executeSuperPlaneWithConfig(t, resolved, core.DefaultHostedLLMModel{
				Provider: models.UsageProviderAnthropic, Model: "claude-sonnet-4-6",
			}, core.HostedLLMAccess{APIKey: "sk-hosted", AllowedModels: []string{"claude-sonnet-4-6"}})
			for _, path := range []string{"prompts/01-implementation.txt", "prompts/03-write-pr.txt", "prompts/04-plan-task.txt"} {
				prompt := requireTaskFile(t, req.Files, path).Content
				if strings.TrimSpace(instructions) == "" {
					assert.NotContains(t, prompt, "Additional instructions")
					continue
				}
				assert.Contains(t, prompt, instructions)
				assert.Equal(t, 1, strings.Count(prompt, instructions))
				assert.Contains(t, prompt, "repository instructions")
				assert.Contains(t, prompt, "managed planning and scoring rules")
				assert.NotContains(t, prompt, "Additional instructions for this implementation")
			}
			if strings.TrimSpace(instructions) != "" {
				context := requireTaskFile(t, req.Files, "planning_additional_instructions.md").Content
				assert.Contains(t, context, instructions)
				assert.Contains(t, context, "managed planning and scoring rules")
			} else {
				assert.False(t, hasTaskFile(req.Files, "planning_additional_instructions.md"))
			}
			assert.Contains(t, requireTaskFile(t, req.Files, "prompts/04-plan-task.txt").Content, planningPrompt)
			assert.Equal(t, "git push", requireTaskFile(t, req.Files, "steps/02-push.sh").Content)
			assert.Equal(t, instructions, resolved["additionalInstructions"])
			assert.Equal(t, "Implement the task", configuration["steps"].([]map[string]any)[0]["prompt"])
		})
	}
}

func TestRunSuperPlaneExecuteDispatchesClaude(t *testing.T) {
	httpContext, _, req := executeSuperPlaneRun(t, map[string]any{
		"machineType": testRunnerMachineType,
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderAnthropic,
		Model:    "claude-sonnet-4-6",
	}, core.HostedLLMAccess{
		APIKey:        "sk-hosted",
		AllowedModels: []string{"claude-sonnet-4-6"},
	})
	assert.Empty(t, httpContext.Requests)

	assert.Equal(t, "sk-hosted", requireEnvironmentValue(t, req.Environment, envAnthropicAPIKey))
	assert.True(t, hasTaskFile(req.Files, "run.js"))
}

func TestRunSuperPlaneExecuteDispatchesCodex(t *testing.T) {
	httpContext, _, req := executeSuperPlaneRun(t, map[string]any{
		"machineType": testRunnerMachineType,
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderOpenAI,
		Model:    "gpt-5",
	}, core.HostedLLMAccess{
		APIKey:        "sk-openai",
		AllowedModels: []string{"gpt-5"},
	})
	assert.Empty(t, httpContext.Requests)

	assert.Equal(t, "sk-openai", requireEnvironmentValue(t, req.Environment, envOpenAIAPIKey))
}

func TestRunSuperPlaneExecuteUsesNodeModelOverDefault(t *testing.T) {
	req := executeSuperPlaneWithConfig(t, map[string]any{
		"machineType": testRunnerMachineType,
		"model":       "openrouter::anthropic/claude-sonnet-4-6",
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderAnthropic,
		Model:    "claude-sonnet-4-6",
	}, core.HostedLLMAccess{
		APIKey:        "sk-or",
		ManagementKey: "sk-or-mgmt",
		AllowedModels: []string{"anthropic/claude-sonnet-4-6", "x-ai/grok-4.6"},
	})
	assert.Equal(t, "sk-or-v1-child", requireEnvironmentValue(t, req.Environment, envOpenRouterAPIKey))
	assert.Equal(t, "3600", requireEnvironmentValue(t, req.Environment, runner.EnvExecutionTimeoutSeconds))
	assert.False(t, hasTaskFile(req.Files, "openrouter_models.json"))
	prepare := requireTaskFile(t, req.Files, "prepare.sh").Content
	assert.Contains(t, prepare, "opencode CLI not found")
	assert.NotContains(t, strings.Join(commandStrings(req.Commands), "\n"), " 128")
}

func TestRunSuperPlaneExecuteMintsOpenRouterChildKey(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	previous := openrouterNowUTC
	openrouterNowUTC = func() time.Time { return now }
	t.Cleanup(func() { openrouterNowUTC = previous })

	httpContext, state, req := executeSuperPlaneRun(t, map[string]any{
		"machineType": testRunnerMachineType,
		"model":       "openrouter::anthropic/claude-sonnet-4-6",
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderAnthropic,
		Model:    "claude-sonnet-4-6",
	}, core.HostedLLMAccess{
		APIKey:        "sk-or",
		ManagementKey: "sk-or-mgmt",
		AllowedModels: []string{"anthropic/claude-sonnet-4-6"},
	})

	require.Len(t, httpContext.Requests, 1)
	mintReq := httpContext.Requests[0]
	assert.Equal(t, http.MethodPost, mintReq.Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys", mintReq.URL.String())
	assert.Equal(t, "Bearer sk-or-mgmt", mintReq.Header.Get("Authorization"))

	mintBody, err := io.ReadAll(mintReq.Body)
	require.NoError(t, err)
	assert.Contains(t, string(mintBody), `"expires_at":"`+openrouterapi.FormatKeyExpiresAt(openrouterapi.ChildKeyExpiresAt(now, 3600))+`"`)
	assert.NotContains(t, string(mintBody), "sk-or-mgmt")
	assert.NotContains(t, string(mintBody), `"sk-or"`)

	assert.Equal(t, "sk-or-v1-child", requireEnvironmentValue(t, req.Environment, envOpenRouterAPIKey))
	for _, variable := range req.Environment {
		assert.NotEqual(t, "sk-or", variable.Value)
		assert.NotEqual(t, "sk-or-mgmt", variable.Value)
	}
	assert.Equal(t, "or-hash-1", state.KVs[runner.OpenRouterChildKeyHashKV])
	assert.NotContains(t, fmt.Sprintf("%v", state.KVs), "sk-or-v1-child")
}

func TestRunSuperPlaneExecuteFailsWhenOpenRouterManagementKeyIsMissing(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}

	httpContext := &contexts.HTTPContext{}
	err := (&RunSuperPlane{}).Execute(core.ExecutionContext{
		RunnerTasks: tasks,
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "openrouter::anthropic/claude-sonnet-4-6",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:    httpContext,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{
				APIKey:        "sk-or",
				AllowedModels: []string{"anthropic/claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrHostedLLMProviderNoManagementKey)
	assert.Empty(t, httpContext.Requests)
}

func TestRunSuperPlaneExecuteFailsWhenOpenRouterCreateKeyErrors(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}

	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Only management keys can perform this operation"}}`))},
	}}
	err := (&RunSuperPlane{}).Execute(core.ExecutionContext{
		RunnerTasks: tasks,
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "openrouter::anthropic/claude-sonnet-4-6",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:    httpContext,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{
				APIKey:        "sk-or",
				ManagementKey: "sk-or-mgmt",
				AllowedModels: []string{"anthropic/claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mint OpenRouter runner key")
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys", httpContext.Requests[0].URL.String())
}

func TestRunSuperPlaneExecuteDeletesChildKeyWhenCreateTaskFails(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{CreateErr: errors.New("task create failed")}

	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"key":"sk-or-v1-child","data":{"hash":"or-hash-1"}}`))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))},
	}}
	err := (&RunSuperPlane{}).Execute(core.ExecutionContext{
		RunnerTasks: tasks,
		Configuration: map[string]any{
			"machineType": testRunnerMachineType,
			"model":       "openrouter::anthropic/claude-sonnet-4-6",
			"steps": []map[string]any{
				{"name": "Hello", "type": "prompt", "prompt": "hello"},
			},
		},
		HTTP:    httpContext,
		Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{
				APIKey:        "sk-or",
				ManagementKey: "sk-or-mgmt",
				AllowedModels: []string{"anthropic/claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create task")
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys", httpContext.Requests[0].URL.String())

	assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[1].URL.String())
	assert.Equal(t, "Bearer sk-or-mgmt", httpContext.Requests[1].Header.Get("Authorization"))
}

func TestBuildSuperPlaneTaskOpenRouterShipsPlanningMCP(t *testing.T) {
	prompt := "hello"
	spec := RunSuperPlaneSpec{
		MachineType: testRunnerMachineType,
		Steps: []runner.AgentStep{
			{Name: "Hello", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
	}
	_, files, err := buildSuperPlaneTask(
		models.UsageProviderOpenRouter,
		spec,
		"anthropic/claude-sonnet-4-6",
		"",
		nil,
		[]runner.TaskEnvironmentVariable{{Name: runner.EnvSuperplanePlanningID, Value: "session-1"}},
		nil,
		nil,
	)
	require.NoError(t, err)
	assert.True(t, hasTaskFile(files, "planning_session_mcp.js"))
	assert.True(t, hasTaskFile(files, "mcp.json"))
	assert.True(t, hasTaskFile(files, "analysis_protocol.js"))
	assert.True(t, hasTaskFile(files, "analysis_protocol_review.md"))
	assert.False(t, hasTaskFile(files, "analysis_protocol.md"))
	assert.False(t, hasTaskFile(files, "openrouter_models.json"))
}

func TestRunSuperPlaneExecuteUsesNodeModelWhenDefaultIsMissing(t *testing.T) {
	req := executeSuperPlaneWithConfig(t, map[string]any{
		"machineType": testRunnerMachineType,
		"model":       "anthropic::claude-sonnet-4-6",
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, core.DefaultHostedLLMModel{}, core.HostedLLMAccess{
		APIKey:        "sk-hosted",
		AllowedModels: []string{"claude-sonnet-4-6"},
	})
	assert.Equal(t, "sk-hosted", requireEnvironmentValue(t, req.Environment, envAnthropicAPIKey))
}

func TestRunSuperPlaneExecuteRejectsMissingDefaultModel(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}
	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		RunnerTasks: tasks,
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
	tasks := &contexts.RunnerTaskContext{}
	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		RunnerTasks: tasks,
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
			CreditErr: billingmodels.ErrHostedCreditEmpty,
			Access: core.HostedLLMAccess{
				APIKey:        "sk-hosted",
				AllowedModels: []string{"claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, billingmodels.ErrHostedCreditEmpty)
}

func executeSuperPlane(t *testing.T, defaultModel core.DefaultHostedLLMModel, access core.HostedLLMAccess) createTaskRequest {
	t.Helper()
	return executeSuperPlaneWithConfig(t, map[string]any{
		"machineType": testRunnerMachineType,
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, defaultModel, access)
}

func executeSuperPlaneWithConfig(
	t *testing.T,
	configuration map[string]any,
	defaultModel core.DefaultHostedLLMModel,
	access core.HostedLLMAccess,
) createTaskRequest {
	t.Helper()
	_, _, req := executeSuperPlaneRun(t, configuration, defaultModel, access)
	return req
}

func executeSuperPlaneRun(
	t *testing.T,
	configuration map[string]any,
	defaultModel core.DefaultHostedLLMModel,
	access core.HostedLLMAccess,
) (*contexts.HTTPContext, *contexts.ExecutionStateContext, createTaskRequest) {
	tasks := &contexts.RunnerTaskContext{}
	t.Helper()

	responses := []*http.Response{}
	if strings.TrimSpace(access.ManagementKey) != "" {
		responses = append([]*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"key":"sk-or-v1-child","data":{"hash":"or-hash-1"}}`))},
		}, responses...)
	}

	httpContext := &contexts.HTTPContext{Responses: responses}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		RunnerTasks:   tasks,
		Configuration: configuration,
		HTTP:          httpContext,
		Secrets:       &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:       &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{
			Default: defaultModel,
			Access:  access,
		},
		ExecutionState: state,
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, tasks.Payload)

	body := tasks.Payload
	require.NotEmpty(t, body)
	var req createTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	return httpContext, state, req
}

func TestRunSuperPlaneExecuteAcceptsPersistedUsageSidecar(t *testing.T) {
	req := executeSuperPlaneWithConfig(t, map[string]any{
		"machineType":    testRunnerMachineType,
		"hostedProvider": models.UsageProviderAnthropic,
		"model":          "claude-sonnet-4-6",
		"credentials":    map[string]any{"source": "hosted"},
		"steps": []map[string]any{
			{"name": "Hello", "type": "prompt", "prompt": "hello"},
		},
	}, core.DefaultHostedLLMModel{
		Provider: models.UsageProviderOpenAI,
		Model:    "gpt-5",
	}, core.HostedLLMAccess{
		APIKey:        "sk-hosted",
		AllowedModels: []string{"claude-sonnet-4-6"},
	})
	assert.Equal(t, "sk-hosted", requireEnvironmentValue(t, req.Environment, envAnthropicAPIKey))
}

func requireEnvironmentValue(t *testing.T, environment []runner.TaskEnvironmentVariable, name string) string {
	t.Helper()
	for _, variable := range environment {
		if variable.Name == name {
			return variable.Value
		}
	}
	t.Fatalf("missing environment %s", name)
	return ""
}

func hasTaskFile(files []runner.TaskFile, path string) bool {
	for _, file := range files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func requireTaskFile(t *testing.T, files []runner.TaskFile, path string) runner.TaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing task file %q", path)
	return runner.TaskFile{}
}

func commandStrings(commands []runner.TaskCommand) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, command.Command)
	}
	return out
}

func TestRunSuperPlaneExecuteRejectsPrivateHostedBaseURL(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{}
	component := &RunSuperPlane{}
	err := component.Execute(core.ExecutionContext{
		RunnerTasks: tasks,
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
