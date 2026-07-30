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

func Test__GetCombinedCommitStatus__Setup(t *testing.T) {
	c := &GetCombinedCommitStatus{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"ref": "main"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing ref", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ref is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "ref": "main"},
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Projects: []ProjectMetadata{{ID: 123, Name: "repo", URL: "http://repo"}},
				},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__GetCombinedCommitStatus__Execute(t *testing.T) {
	c := &GetCombinedCommitStatus{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"groupId":     "123",
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	decodeCombined := func(t *testing.T, executionState *contexts.ExecutionStateContext) CombinedCommitStatus {
		require.Len(t, executionState.Payloads, 1)
		assert.Equal(t, CombinedCommitStatusPayloadType, executionState.Type)
		payload := executionState.Payloads[0].(map[string]any)
		var combined CombinedCommitStatus
		payloadBytes, _ := json.Marshal(payload["data"])
		require.NoError(t, json.Unmarshal(payloadBytes, &combined))
		return combined
	}

	t.Run("rolls up to the worst state", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"id": 93, "sha": "abc123", "ref": "main", "status": "success", "name": "ci/superplane"},
					{"id": 92, "sha": "abc123", "ref": "main", "status": "failed", "name": "lint"}
				]`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"project": "123", "ref": "main"},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		combined := decodeCombined(t, executionState)
		assert.Equal(t, "failed", combined.State)
		assert.Equal(t, "abc123", combined.SHA)
		assert.Equal(t, 2, combined.TotalCount)
		require.Len(t, combined.Statuses, 2)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/repository/commits/main/statuses?page=1&per_page=100", httpCtx.Requests[0].URL.String())
	})

	t.Run("all success rolls up to success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"id": 93, "sha": "abc123", "status": "success", "name": "ci"},
					{"id": 92, "sha": "abc123", "status": "success", "name": "lint"}
				]`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"project": "123", "ref": "abc123"},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)
		assert.Equal(t, "success", decodeCombined(t, executionState).State)
	})

	t.Run("no statuses yields empty state and empty array", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{GitlabMockResponse(http.StatusOK, `[]`)},
		}
		ctx := core.ExecutionContext{
			Configuration:  map[string]any{"project": "123", "ref": "main"},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		combined := decodeCombined(t, executionState)
		assert.Equal(t, "", combined.State)
		assert.Equal(t, 0, combined.TotalCount)

		payload := executionState.Payloads[0].(map[string]any)
		dataBytes, _ := json.Marshal(payload["data"])
		assert.Contains(t, string(dataBytes), `"statuses":[]`)
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{"project": "123", "ref": "main"},
			Integration:   integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Commit Not Found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get combined commit status")
	})
}
