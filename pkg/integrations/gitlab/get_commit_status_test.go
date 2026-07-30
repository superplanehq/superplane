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

	t.Run("emits the commit with its overall status and last pipeline", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": "18f3e63d05582537db6d183d9d557be09e1f90c8",
					"short_id": "18f3e63d",
					"title": "feat: add health endpoint",
					"status": "failed",
					"web_url": "https://gitlab.com/g/p/-/commit/18f3e63d05582537db6d183d9d557be09e1f90c8",
					"last_pipeline": {"id": 1024, "ref": "main", "sha": "18f3e63d05582537db6d183d9d557be09e1f90c8", "status": "failed", "web_url": "https://gitlab.com/g/p/-/pipelines/1024"}
				}`),
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

		require.Len(t, executionState.Payloads, 1)
		assert.Equal(t, CommitPayloadType, executionState.Type)

		payload := executionState.Payloads[0].(map[string]any)
		var commit Commit
		payloadBytes, _ := json.Marshal(payload["data"])
		require.NoError(t, json.Unmarshal(payloadBytes, &commit))
		assert.Equal(t, "failed", commit.Status)
		require.NotNil(t, commit.LastPipeline)
		assert.Equal(t, "failed", commit.LastPipeline.Status)
		assert.Equal(t, "https://gitlab.com/g/p/-/pipelines/1024", commit.LastPipeline.WebURL)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/repository/commits/main", httpCtx.Requests[0].URL.String())
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
		assert.Contains(t, err.Error(), "failed to get commit status")
	})
}
