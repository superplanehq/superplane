package coolify

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

func Test__Coolify_ListApplications__Execute(t *testing.T) {
	component := &ListApplications{}

	t.Run("emits applications payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[
							{"uuid":"abc123","name":"frontend","status":"running","fqdn":"https://app.example.com","git_repository":"https://github.com/example/frontend","git_branch":"main"},
							{"uuid":"def456","name":"api","status":"exited"}
						]`,
					)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, ListApplicationsPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emitted := readMap(executionState.Payloads[0])
		data := readMap(emitted["data"])
		assert.Equal(t, 2, data["count"])

		applications, ok := data["applications"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, applications, 2)
		assert.Equal(t, "abc123", applications[0]["uuid"])
		assert.Equal(t, "frontend", applications[0]["name"])
		assert.Equal(t, "running", applications[0]["status"])
		assert.Equal(t, "https://github.com/example/frontend", applications[0]["gitRepository"])
		assert.Equal(t, "main", applications[0]["gitBranch"])
		assert.Equal(t, "def456", applications[1]["uuid"])
		assert.NotContains(t, applications[1], "fqdn", "empty optional fields should be omitted")
		assert.NotContains(t, applications[1], "gitRepository")

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://coolify.example.com/api/v1/applications", httpCtx.Requests[0].URL.String())
	})

	t.Run("API error -> propagated", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message":"boom"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})
}

// readMap is a small helper used across the coolify tests to unwrap the
// emitted payload envelope produced by the ExecutionStateContext mock.
func readMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return item
}
