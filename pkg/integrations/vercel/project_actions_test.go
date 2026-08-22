package vercel

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Vercel_GetProject(t *testing.T) {
	t.Run("missing project -> error", func(t *testing.T) {
		err := (&GetProject{}).Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "project is required")
	})

	t.Run("valid configuration -> emits project payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"id":"prj_1","name":"my-app","framework":"nextjs","createdAt":1735689600000}`,
		)}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&GetProject{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "prj_1"},
		})

		require.NoError(t, err)
		assert.Equal(t, ProjectPayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "prj_1", data["projectId"])
		assert.Equal(t, "nextjs", data["framework"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v9/projects/prj_1", request.URL.Path)
	})
}

func Test__Vercel_CreateProject(t *testing.T) {
	t.Run("missing name -> error", func(t *testing.T) {
		err := (&CreateProject{}).Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid configuration -> creates project and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"id":"prj_new","name":"tenant-1234"}`,
		)}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&CreateProject{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"name": "tenant-1234", "framework": "NextJS"},
		})

		require.NoError(t, err)
		assert.Equal(t, ProjectPayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "tenant-1234", data["name"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v11/projects", request.URL.Path)

		body, bodyErr := io.ReadAll(request.Body)
		require.NoError(t, bodyErr)
		decodedBody := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &decodedBody))
		assert.Equal(t, "tenant-1234", decodedBody["name"])
		assert.Equal(t, "nextjs", decodedBody["framework"], "framework preset is normalized to lowercase")
	})
}

func Test__Vercel_UpsertEnvVar(t *testing.T) {
	t.Run("missing key -> error", func(t *testing.T) {
		err := (&UpsertEnvVar{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1"},
		})
		require.ErrorContains(t, err, "key is required")
	})

	t.Run("invalid environment -> error", func(t *testing.T) {
		err := (&UpsertEnvVar{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1", "key": "API_URL", "targets": []string{"staging"}},
		})
		require.ErrorContains(t, err, "environments must be one of")
	})

	t.Run("valid configuration -> upserts variable", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"created":{"key":"API_URL","type":"encrypted","target":["production"],"id":"env_1"},"failed":[]}`,
		)}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&UpsertEnvVar{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"project": "prj_1",
				"key":     "API_URL",
				"value":   "https://api.example.com",
				"targets": []string{"production", "preview"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, EnvVarPayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "API_URL", data["key"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v10/projects/prj_1/env", request.URL.Path)
		assert.Equal(t, "true", request.URL.Query().Get("upsert"), "upsert makes the component create or update")

		body, bodyErr := io.ReadAll(request.Body)
		require.NoError(t, bodyErr)
		decodedBody := map[string]any{}
		require.NoError(t, json.Unmarshal(body, &decodedBody))
		assert.Equal(t, "API_URL", decodedBody["key"])
		assert.Equal(t, "encrypted", decodedBody["type"], "defaults to encrypted storage")

		targets, ok := decodedBody["target"].([]any)
		require.True(t, ok)
		assert.Len(t, targets, 2)
	})

	t.Run("failed upsert -> error with message", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"failed":[{"error":{"code":"ENV_VAR_FAILED","message":"variable is protected"}}]}`,
		)}}

		err := (&UpsertEnvVar{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"project": "prj_1", "key": "LOCKED", "value": "x"},
		})

		require.ErrorContains(t, err, "variable is protected")
	})
}

func Test__Vercel_Domains(t *testing.T) {
	t.Run("add missing domain -> error", func(t *testing.T) {
		err := (&AddDomain{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1"},
		})
		require.ErrorContains(t, err, "domain is required")
	})

	t.Run("remove missing domain -> error", func(t *testing.T) {
		err := (&RemoveDomain{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1"},
		})
		require.ErrorContains(t, err, "domain is required")
	})

	t.Run("add domain -> emits domain payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"name":"www.example.com","verified":false}`,
		)}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&AddDomain{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "prj_1", "domain": "WWW.Example.com"},
		})

		require.NoError(t, err)
		assert.Equal(t, DomainPayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "www.example.com", data["name"], "domain names are normalized to lowercase")
		assert.Equal(t, false, data["verified"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v10/projects/prj_1/domains", request.URL.Path)
	})

	t.Run("remove domain -> emits confirmation", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, "")}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&RemoveDomain{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "prj_1", "domain": "www.example.com"},
		})

		require.NoError(t, err)
		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, true, data["removed"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodDelete, request.Method)
		assert.Equal(t, "/v9/projects/prj_1/domains/www.example.com", request.URL.Path)
	})
}
