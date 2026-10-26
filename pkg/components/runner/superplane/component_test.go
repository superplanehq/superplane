package superplane

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
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
	assert.True(t, strings.HasPrefix(requireEnvironmentValue(t, req.Environment, envOpenRouterAPIKey), "pending-runner-key-"))
	assert.Equal(t, "3600", requireEnvironmentValue(t, req.Environment, runner.EnvExecutionTimeoutSeconds))
	assert.False(t, hasTaskFile(req.Files, "openrouter_models.json"))
	prepare := requireTaskFile(t, req.Files, "prepare.sh").Content
	assert.Contains(t, prepare, "opencode CLI not found")
	assert.NotContains(t, strings.Join(commandStrings(req.Commands), "\n"), " 128")
}

func TestRunSuperPlaneQueuesOpenRouterWithoutMintingProviderKey(t *testing.T) {
	tasks := &recordingRunnerTaskContext{RunnerTaskContext: &contexts.RunnerTaskContext{}}
	httpContext := &contexts.HTTPContext{}
	providerID := uuid.NewString()
	err := (&RunSuperPlane{}).Execute(core.ExecutionContext{
		OrganizationID: uuid.NewString(), RunID: uuid.New(), RunnerTasks: tasks,
		Configuration: map[string]any{"machineType": testRunnerMachineType,
			"model": "openrouter::anthropic/claude-sonnet-4-6",
			"steps": []map[string]any{{"name": "Hello", "type": "prompt", "prompt": "hello"}}},
		HTTP: httpContext, Secrets: &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook: &contexts.NodeWebhookContext{},
		HostedLLM: &contexts.HostedLLMContext{Access: core.HostedLLMAccess{
			ResourceID: providerID, APIKey: "provider-key", ManagementKey: "management-key",
			AllowedModels: []string{"anthropic/claude-sonnet-4-6"}}},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}}, Requests: &contexts.RequestContext{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, tasks.CreatedID)
	require.Empty(t, httpContext.Requests, "provider creation belongs to runner preparation after reservation")
	var request createTaskRequest
	require.NoError(t, json.Unmarshal(tasks.Payload, &request))
	pending := requireEnvironmentValue(t, request.Environment, envOpenRouterAPIKey)
	require.True(t, strings.HasPrefix(pending, "pending-runner-key-"))
	require.NotContains(t, string(tasks.Payload), "provider-key")
	require.NotContains(t, string(tasks.Payload), "management-key")
	require.Len(t, tasks.refreshes, 1)
	require.Equal(t, "hosted_provider", tasks.refreshes[0].Kind)
	require.Equal(t, providerID, tasks.refreshes[0].ResourceID)
	require.Equal(t, models.UsageProviderOpenRouter, tasks.refreshes[0].Recipient)
	require.Equal(t, pending, tasks.refreshes[0].Current)
}

type recordingRunnerTaskContext struct {
	*contexts.RunnerTaskContext
	refreshes []runnerapi.CredentialRefresh
}

func (c *recordingRunnerTaskContext) RecordCredentialRefresh(refresh runnerapi.CredentialRefresh) {
	c.refreshes = append(c.refreshes, refresh)
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

func TestRunSuperPlaneDoesNotMintWhenQueueCreationFails(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{CreateErr: errors.New("task create failed")}

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
				ManagementKey: "sk-or-mgmt",
				AllowedModels: []string{"anthropic/claude-sonnet-4-6"},
			},
		},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create task")
	assert.Empty(t, httpContext.Requests)
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

	httpContext := &contexts.HTTPContext{}
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
