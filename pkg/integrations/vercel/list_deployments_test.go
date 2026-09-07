package vercel

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func Test__Vercel_ListDeployments(t *testing.T) {
	t.Run("invalid target -> error", func(t *testing.T) {
		err := (&ListDeployments{}).Setup(core.SetupContext{
			Configuration: map[string]any{"target": "staging"},
		})
		require.ErrorContains(t, err, "target must be one of")
	})

	t.Run("invalid state -> error", func(t *testing.T) {
		err := (&ListDeployments{}).Setup(core.SetupContext{
			Configuration: map[string]any{"state": "DONE"},
		})
		require.ErrorContains(t, err, "state must be one of")
	})

	t.Run("limit above 100 -> error", func(t *testing.T) {
		err := (&ListDeployments{}).Setup(core.SetupContext{
			Configuration: map[string]any{"limit": 200},
		})
		require.ErrorContains(t, err, "limit must be between 0 and 100")
	})

	t.Run("valid configuration -> emits deployments", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"deployments":[{"uid":"dpl_1","name":"my-app","url":"my-app.vercel.app","readyState":"READY","target":"production","projectId":"prj_1","createdAt":1755850000000}],"pagination":{"count":1}}`,
		)}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&ListDeployments{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "prj_1", "target": "production", "state": "READY", "limit": 50},
		})

		require.NoError(t, err)
		assert.Equal(t, ListDeploymentsPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, 1, data["count"])

		deployments, ok := data["deployments"].([]map[string]any)
		require.True(t, ok)
		assert.Equal(t, "dpl_1", deployments[0]["deploymentId"], "uid from list responses becomes deploymentId")

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v7/deployments", request.URL.Path)
		query := request.URL.Query()
		assert.Equal(t, "prj_1", query.Get("projectId"))
		assert.Equal(t, "production", query.Get("target"))
		assert.Equal(t, "READY", query.Get("state"))
		assert.Equal(t, "50", query.Get("limit"))
	})
}

func Test__Vercel_CancelDeployment(t *testing.T) {
	t.Run("missing deploymentId -> error", func(t *testing.T) {
		err := (&CancelDeployment{}).Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "deploymentId is required")
	})

	t.Run("valid configuration -> emits canceled deployment", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK,
			`{"id":"dpl_1","readyState":"CANCELED","name":"my-app"}`,
		)}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&CancelDeployment{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"deploymentId": "dpl_1"},
		})

		require.NoError(t, err)
		assert.Equal(t, GetDeploymentPayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "CANCELED", data["readyState"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPatch, request.Method)
		assert.Equal(t, "/v12/deployments/dpl_1/cancel", request.URL.Path)
	})
}

func Test__Vercel_Rollback(t *testing.T) {
	t.Run("missing deploymentId -> error", func(t *testing.T) {
		err := (&RollbackProduction{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1"},
		})
		require.ErrorContains(t, err, "deploymentId is required")
	})

	t.Run("valid configuration -> rolls back and emits confirmation", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusCreated, "")}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&RollbackProduction{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "prj_1", "deploymentId": "dpl_good", "description": "bad release"},
		})

		require.NoError(t, err)
		assert.Equal(t, RollbackPayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, true, data["rolledBack"])
		assert.Equal(t, "dpl_good", data["deploymentId"])

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v1/projects/prj_1/rollback/dpl_good", request.URL.Path)
		assert.Equal(t, "bad release", request.URL.Query().Get("description"))
	})
}
