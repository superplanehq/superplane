package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetCommitStatus__Setup(t *testing.T) {
	c := &GetCommitStatus{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"sha": "abc123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing sha", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commit SHA is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
			},
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Projects: []ProjectMetadata{
						{ID: 123, Name: "repo", URL: "http://repo"},
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__GetCommitStatus__Execute(t *testing.T) {
	c := &GetCommitStatus{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"groupId":     "123",
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"id": 93, "sha": "abc123", "ref": "main", "status": "success", "name": "ci/superplane", "target_url": "https://ci.example.com/runs/42"},
					{"id": 92, "sha": "abc123", "ref": "main", "status": "failed", "name": "lint"}
				]`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
			},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, CommitStatusesPayloadType, executionState.Type)

		payload := executionState.Payloads[0].(map[string]any)
		var statuses []CommitStatus
		payloadBytes, _ := json.Marshal(payload["data"])
		require.NoError(t, json.Unmarshal(payloadBytes, &statuses))
		require.Len(t, statuses, 2)
		assert.Equal(t, "success", statuses[0].Status)
		assert.Equal(t, "ci/superplane", statuses[0].Name)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/repository/commits/abc123/statuses?order_by=id&page=1&per_page=100&sort=desc", httpCtx.Requests[0].URL.String())
	})

	t.Run("emits an empty array when there are no statuses", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[]`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"project": "123", "sha": "abc123"},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		dataBytes, err := json.Marshal(payload["data"])
		require.NoError(t, err)
		assert.Equal(t, "[]", string(dataBytes))
	})

	t.Run("applies ref and name filters", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[{"id": 93, "sha": "abc123", "status": "success", "name": "ci/superplane"}]`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
				"ref":     "main",
				"name":    "ci/superplane",
			},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/repository/commits/abc123/statuses?name=ci%2Fsuperplane&order_by=id&page=1&per_page=100&ref=main&sort=desc", httpCtx.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Commit Not Found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit status")
	})
}
