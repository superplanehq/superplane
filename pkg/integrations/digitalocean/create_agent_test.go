package digitalocean

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateAgent__Setup(t *testing.T) {
	component := &CreateAgent{}

	baseConfig := func(overrides map[string]any) map[string]any {
		cfg := map[string]any{
			"name":            "my-agent",
			"instruction":     "You are a helpful assistant",
			"modelUUID":       "test-model-uuid",
			"workspaceSource": "existing",
			"workspaceUUID":   "test-workspace-uuid",
			"region":          "tor1",
		}
		for k, v := range overrides {
			if v == nil {
				delete(cfg, k)
			} else {
				cfg[k] = v
			}
		}
		return cfg
	}

	t.Run("missing name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{"name": nil})})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing instruction -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{"instruction": nil})})
		require.ErrorContains(t, err, "instruction is required")
	})

	t.Run("missing modelUUID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{"modelUUID": nil})})
		require.ErrorContains(t, err, "model is required")
	})

	t.Run("existing workspace without UUID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{"workspaceUUID": nil})})
		require.ErrorContains(t, err, "workspace is required")
	})

	t.Run("new workspace without name -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{
			"workspaceSource": "new",
			"workspaceUUID":   nil,
		})})
		require.ErrorContains(t, err, "workspace name is required")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{"region": nil})})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("valid config with existing workspace -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(nil)})
		require.NoError(t, err)
	})

	t.Run("valid config with new workspace -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: baseConfig(map[string]any{
			"workspaceSource": "new",
			"workspaceUUID":   nil,
			"workspaceName":   "my-workspace",
		})})
		require.NoError(t, err)
	})
}

func Test__CreateAgent__Execute(t *testing.T) {
	component := &CreateAgent{}

	integrationCtx := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}
	}

	baseConfig := map[string]any{
		"name":               "my-agent",
		"instruction":        "You are a helpful assistant",
		"modelUUID":          "test-model-uuid",
		"workspaceSource":    "existing",
		"workspaceUUID":      "test-workspace-uuid",
		"region":             "tor1",
		"projectID":          "test-project-id",
		"useDefaultSettings": true,
	}

	t.Run("success: existing workspace, no provider key, default settings", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// 1. POST /v2/gen-ai/agents (includes workspace_uuid)
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"agent": {"uuid": "agent-uuid-123", "name": "my-agent"}
				}`))},
				// 2. PUT /v2/gen-ai/agents/{uuid}/deployment
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  baseConfig,
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Verify exactly 2 HTTP requests: create agent + deploy
		require.Len(t, httpCtx.Requests, 2)

		// Check agent create request — workspace_uuid must be embedded
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://api.digitalocean.com/v2/gen-ai/agents", httpCtx.Requests[0].URL.String())
		reqBody, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var createReq map[string]any
		require.NoError(t, json.Unmarshal(reqBody, &createReq))
		assert.Equal(t, "my-agent", createReq["name"])
		assert.Equal(t, "test-model-uuid", createReq["model_uuid"])
		assert.Equal(t, "tor1", createReq["region"])
		assert.Equal(t, "test-project-id", createReq["project_id"])
		assert.Equal(t, "test-workspace-uuid", createReq["workspace_uuid"], "workspace_uuid must be in create request")
		assert.Nil(t, createReq["anthropic_key_uuid"], "should not send provider key when not configured")

		// Check deployment request
		assert.Equal(t, http.MethodPut, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/agents/agent-uuid-123/deployment")

		// Verify metadata stored
		metadata, ok := metadataCtx.Metadata.(CreateAgentExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "agent-uuid-123", metadata.AgentUUID)
		assert.NotZero(t, metadata.StartedAt)

		// Verify poll scheduled, not yet emitting
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, createAgentPollInterval, requestCtx.Duration)
		assert.False(t, executionState.Passed)
	})

	t.Run("success: creates new workspace first, then agent", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// 1. POST /v2/gen-ai/workspaces
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"workspace": {"uuid": "new-ws-uuid", "name": "my-workspace"}
				}`))},
				// 2. POST /v2/gen-ai/agents (workspace_uuid = new-ws-uuid)
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"agent": {"uuid": "agent-uuid-123", "name": "my-agent"}
				}`))},
				// 3. PUT /v2/gen-ai/agents/{uuid}/deployment
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":               "my-agent",
				"instruction":        "You are a helpful assistant",
				"modelUUID":          "test-model-uuid",
				"workspaceSource":    "new",
				"workspaceName":      "my-workspace",
				"region":             "tor1",
				"projectID":          "test-project-id",
				"useDefaultSettings": true,
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 3)

		// First request: create workspace
		assert.Equal(t, "https://api.digitalocean.com/v2/gen-ai/workspaces", httpCtx.Requests[0].URL.String())
		wsBody, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var wsReq map[string]any
		require.NoError(t, json.Unmarshal(wsBody, &wsReq))
		assert.Equal(t, "my-workspace", wsReq["name"])

		// Second request: create agent with new workspace UUID embedded
		agentBody, _ := io.ReadAll(httpCtx.Requests[1].Body)
		var agentReq map[string]any
		require.NoError(t, json.Unmarshal(agentBody, &agentReq))
		assert.Equal(t, "new-ws-uuid", agentReq["workspace_uuid"], "new workspace UUID must be passed in create request")
	})

	t.Run("success: Anthropic provider key -> registers key first, includes uuid in agent request", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// 1. POST /v2/gen-ai/anthropic/keys (provider from spec.ModelProvider, no extra GET)
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"api_key_info": {"uuid": "ant-key-uuid", "name": "my-agent"}
				}`))},
				// 2. POST /v2/gen-ai/agents
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"agent": {"uuid": "agent-uuid-123", "name": "my-agent"}
				}`))},
				// 3. PUT /v2/gen-ai/agents/{uuid}/deployment
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":               "my-agent",
				"instruction":        "You are a helpful assistant",
				"modelProvider":      "anthropic",
				"modelUUID":          "anthropic-model-uuid",
				"providerAPIKey":     "sk-ant-test-key",
				"workspaceSource":    "existing",
				"workspaceUUID":      "test-workspace-uuid",
				"region":             "tor1",
				"projectID":          "test-project-id",
				"useDefaultSettings": true,
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		// 3 requests: POST anthropic/keys, POST agents, PUT deployment
		// (no extra GET /models — provider is read directly from spec.ModelProvider)
		require.Len(t, httpCtx.Requests, 3)

		// First request: register Anthropic key
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/anthropic/keys")

		// Second request: create agent — must include anthropic_key_uuid and workspace_uuid.
		// model_provider_key_uuid must NOT be set: it is a separate resource type
		// (from /v2/gen-ai/model_provider_keys) and using an anthropic key UUID
		// would cause a 404.
		agentBody, _ := io.ReadAll(httpCtx.Requests[1].Body)
		var agentReq map[string]any
		require.NoError(t, json.Unmarshal(agentBody, &agentReq))
		assert.Equal(t, "ant-key-uuid", agentReq["anthropic_key_uuid"])
		assert.Nil(t, agentReq["model_provider_key_uuid"], "model_provider_key_uuid must not be set with anthropic key UUID")
		assert.Equal(t, "test-workspace-uuid", agentReq["workspace_uuid"])
	})

	t.Run("success: OpenAI provider key -> registers key first, includes uuid in agent request", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// 1. POST /v2/gen-ai/openai/keys
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"api_key_info": {"uuid": "oai-key-uuid", "name": "my-agent"}
				}`))},
				// 2. POST /v2/gen-ai/agents
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"agent": {"uuid": "agent-uuid-456", "name": "my-agent"}
				}`))},
				// 3. PUT /v2/gen-ai/agents/{uuid}/deployment
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":               "my-agent",
				"instruction":        "You are a helpful assistant",
				"modelProvider":      "openai",
				"modelUUID":          "openai-model-uuid",
				"providerAPIKey":     "sk-oai-test-key",
				"workspaceSource":    "existing",
				"workspaceUUID":      "test-workspace-uuid",
				"region":             "tor1",
				"projectID":          "test-project-id",
				"useDefaultSettings": true,
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 3)

		// First request: register OpenAI key
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/openai/keys")

		// Second request: create agent — must include open_ai_key_uuid and workspace_uuid.
		// model_provider_key_uuid must NOT be set: it is a separate resource type
		// (from /v2/gen-ai/model_provider_keys) and using an openai key UUID
		// would cause a 404.
		agentBody, _ := io.ReadAll(httpCtx.Requests[1].Body)
		var agentReq map[string]any
		require.NoError(t, json.Unmarshal(agentBody, &agentReq))
		assert.Equal(t, "oai-key-uuid", agentReq["open_ai_key_uuid"])
		assert.Nil(t, agentReq["model_provider_key_uuid"], "model_provider_key_uuid must not be set with openai key UUID")
		assert.Equal(t, "test-workspace-uuid", agentReq["workspace_uuid"])
	})

	t.Run("DO API 404 -> returns error with body", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"id":"not_found","message":"failed to create agent"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  baseConfig,
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create agent")
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("DO API 403 -> returns error with body", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"id":"forbidden","message":"failed to create agent"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  baseConfig,
			HTTP:           httpCtx,
			Integration:    integrationCtx(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("nil integration -> returns clear error (not panic)", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  baseConfig,
			HTTP:           &contexts.HTTPContext{},
			Integration:    nil,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "integration is not configured")
	})
}

func Test__CreateAgent__HandleAction_Poll(t *testing.T) {
	component := &CreateAgent{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "test-token"},
	}

	activeMetadata := func() *contexts.MetadataContext {
		return &contexts.MetadataContext{
			Metadata: map[string]any{
				"agentUUID": "test-agent-uuid",
				"startedAt": time.Now().UnixNano(),
			},
		}
	}

	t.Run("deployment running -> creates API key and emits", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GET /v2/gen-ai/agents/{uuid}
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"agent": {
						"uuid": "test-agent-uuid",
						"name": "my-agent",
						"url": "https://agent.example.com",
						"deployment": {
							"uuid": "dep-uuid",
							"status": "DEPLOYMENT_STATUS_RUNNING",
							"url": "https://agent.example.com"
						}
					}
				}`))},
				// POST /v2/gen-ai/agents/{uuid}/api_keys
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{
					"api_key_info": {
						"uuid": "key-uuid",
						"name": "my-agent-key",
						"secret_key": "sk-agent-secret-xyz"
					}
				}`))},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			Metadata:       activeMetadata(),
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, agentPayloadType, executionState.Type)
		require.Len(t, httpCtx.Requests, 2)
		assert.Contains(t, httpCtx.Requests[1].URL.String(), "/api_keys")
	})

	t.Run("deployment pending -> schedules another poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"agent": {
						"uuid": "test-agent-uuid",
						"deployment": {"uuid": "dep-uuid", "status": "DEPLOYMENT_STATUS_PENDING"}
					}
				}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			Metadata:       activeMetadata(),
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, createAgentPollInterval, requestCtx.Duration)
	})

	t.Run("no deployment yet -> schedules another poll", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"agent": {"uuid": "test-agent-uuid", "name": "my-agent"}
				}`))},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			Metadata:       activeMetadata(),
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("deployment error -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"agent": {
						"uuid": "test-agent-uuid",
						"deployment": {"uuid": "dep-uuid", "status": "DEPLOYMENT_STATUS_ERROR"}
					}
				}`))},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			Metadata:       activeMetadata(),
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to deploy")
		assert.False(t, executionState.Passed)
	})

	t.Run("timed out -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		pastStart := time.Now().Add(-(createAgentTimeout + time.Minute)).UnixNano()

		err := component.HandleAction(core.ActionContext{
			Name:        "poll",
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"agentUUID": "test-agent-uuid",
					"startedAt": pastStart,
				},
			},
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}
