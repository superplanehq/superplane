package gitlab

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

func Test__PublishCommitStatus__Setup(t *testing.T) {
	c := &PublishCommitStatus{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"sha":   "abc123",
				"state": "success",
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
				"state":   "success",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commit SHA is required")
	})

	t.Run("invalid state", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
				"state":   "bogus",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
				"state":   "success",
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

func Test__PublishCommitStatus__Execute(t *testing.T) {
	c := &PublishCommitStatus{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"groupId":     "123",
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	t.Run("success with optional fields", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 93,
					"sha": "18f3e63d05582537db6d183d9d557be09e1f90c8",
					"ref": "main",
					"status": "success",
					"name": "ci/superplane",
					"target_url": "https://ci.example.com/runs/42",
					"description": "Build passed",
					"created_at": "2026-02-13T18:00:00.000Z",
					"allow_failure": false,
					"coverage": 92.5,
					"pipeline_id": 1024
				}`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "123",
				"sha":         "18f3e63d05582537db6d183d9d557be09e1f90c8",
				"state":       "success",
				"name":        "ci/superplane",
				"targetUrl":   "https://ci.example.com/runs/42",
				"description": "Build passed",
				"coverage":    92.5,
			},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, CommitStatusPayloadType, executionState.Type)

		payload := executionState.Payloads[0].(map[string]any)
		var status CommitStatus
		payloadBytes, _ := json.Marshal(payload["data"])
		require.NoError(t, json.Unmarshal(payloadBytes, &status))
		assert.Equal(t, 93, status.ID)
		assert.Equal(t, "success", status.Status)
		assert.Equal(t, "ci/superplane", status.Name)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/statuses/18f3e63d05582537db6d183d9d557be09e1f90c8", httpCtx.Requests[0].URL.String())

		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		bodyString := string(body)
		assert.Contains(t, bodyString, `"state":"success"`)
		assert.Contains(t, bodyString, `"name":"ci/superplane"`)
		assert.Contains(t, bodyString, `"coverage":92.5`)
	})

	t.Run("omits unset optional fields", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{"id": 94, "sha": "abc123", "status": "pending"}`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
				"state":   "pending",
			},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		bodyString := string(body)
		assert.Contains(t, bodyString, `"state":"pending"`)
		assert.NotContains(t, bodyString, `"name"`)
		assert.NotContains(t, bodyString, `"coverage"`)
		assert.NotContains(t, bodyString, `"target_url"`)
	})

	t.Run("trims name and ref before sending", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{"id": 95, "sha": "abc123", "status": "success"}`),
			},
		}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
				"state":   "success",
				"name":    "  ci/superplane  ",
				"ref":     "  main  ",
			},
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		bodyString := string(body)
		assert.Contains(t, bodyString, `"name":"ci/superplane"`)
		assert.Contains(t, bodyString, `"ref":"main"`)
	})

	t.Run("requires sha at execution time", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "  ",
				"state":   "success",
			},
			Integration: integration,
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commit SHA is required")
	})

	t.Run("invalid pipeline ID", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":    "123",
				"sha":        "abc123",
				"state":      "success",
				"pipelineId": "not-a-number",
			},
			Integration: integration,
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pipeline ID")
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"sha":     "abc123",
				"state":   "failed",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusBadRequest, `{"message": "400 Bad Request"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to publish commit status")
	})
}
