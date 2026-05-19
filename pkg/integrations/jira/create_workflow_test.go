package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateWorkflow__Setup(t *testing.T) {
	component := CreateWorkflow{}

	t.Run("missing status -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"name":        "Support workflow",
				"transitions": []map[string]any{{"name": "Start", "from": []string{"To Do"}, "to": "Done"}},
			},
		})

		require.ErrorContains(t, err, "at least one status is required")
	})

	t.Run("project scope requires project", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"name":     "Support workflow",
				"scope":    workflowScopeProject,
				"statuses": []map[string]any{{"name": "To Do", "category": "TODO"}},
				"transitions": []map[string]any{
					{"name": "Start", "from": []string{"To Do"}, "to": "To Do"},
				},
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("unknown transition status -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"name":     "Support workflow",
				"statuses": []map[string]any{{"name": "To Do", "category": "TODO"}},
				"transitions": []map[string]any{
					{"name": "Start", "from": []string{"To Do"}, "to": "Done"},
				},
			},
		})

		require.ErrorContains(t, err, "unknown status")
	})

	t.Run("valid setup stores workflow metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"name": "Support workflow",
				"statuses": []map[string]any{
					{"name": "To Do", "category": "TODO"},
					{"name": "Done", "category": "DONE"},
				},
				"transitions": []map[string]any{
					{"name": "Complete", "from": []string{"To Do"}, "to": "Done"},
				},
			},
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Support workflow", nodeMetadata.WorkflowName)
	})
}

func Test__CreateWorkflow__Execute(t *testing.T) {
	component := CreateWorkflow{}

	t.Run("creates workflow and emits first created workflow", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"workflows":[{"id":"wf-1","name":"Support workflow","version":{"id":"v-1","versionNumber":1}}],
						"statuses":[]
					}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":        "Support workflow",
				"description": "Request lifecycle",
				"statuses": []map[string]any{
					{"name": "To Do", "category": "TODO"},
					{"name": "In Progress", "category": "IN_PROGRESS"},
					{"name": "Done", "category": "DONE"},
				},
				"transitions": []map[string]any{
					{"name": "Start work", "from": []string{"To Do"}, "to": "In Progress", "type": "directed"},
					{"name": "Close", "from": []string{"any"}, "to": "Done", "type": "global"},
				},
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, CreateWorkflowPayloadType, execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/workflows/create")

		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, workflowScopeGlobal, payload["scope"].(map[string]any)["type"])
		assert.Len(t, payload["statuses"].([]any), 3)
		transitions := payload["workflows"].([]any)[0].(map[string]any)["transitions"].([]any)
		assert.Equal(t, "INITIAL", transitions[0].(map[string]any)["type"])
	})

	t.Run("project-scoped workflow includes project id in scope", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test"}]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"workflows":[{"id":"wf-2","name":"Scoped workflow","version":{"id":"v-2","versionNumber":1}}],
						"statuses":[]
					}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":     "Scoped workflow",
				"scope":    workflowScopeProject,
				"project":  "TEST",
				"statuses": []map[string]any{{"name": "To Do", "category": "TODO"}},
				"transitions": []map[string]any{
					{"name": "Loop", "from": []string{"To Do"}, "to": "To Do"},
				},
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)

		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		scope := payload["scope"].(map[string]any)
		assert.Equal(t, workflowScopeProject, scope["type"])
		assert.Equal(t, "10000", scope["project"].(map[string]any)["id"])
	})

	t.Run("permission denied surfaces admin hint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Administer Jira required"]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":     "Support workflow",
				"statuses": []map[string]any{{"name": "To Do", "category": "TODO"}},
				"transitions": []map[string]any{
					{"name": "Loop", "from": []string{"To Do"}, "to": "To Do"},
				},
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Jira admin")
	})
}
