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

func Test__AssignWorkflowToProject__Setup(t *testing.T) {
	component := AssignWorkflowToProject{}

	t.Run("missing workflow scheme -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "TEST"},
		})

		require.ErrorContains(t, err, "workflowScheme is required")
	})

	t.Run("team-managed project -> clear error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"TEAM","name":"Team","style":"next-gen","simplified":true}`)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			HTTP:          httpContext,
			Integration:   newAuthorizedIntegration(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "TEAM", "workflowScheme": "101010"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "company-managed projects")
	})

	t.Run("valid setup stores workflow scheme metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"TEST","name":"Test","style":"classic","simplified":false}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"isLast":true,"values":[{"id":101010,"name":"Support scheme"}]}`)),
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			HTTP:          httpContext,
			Integration:   newAuthorizedIntegration(),
			Metadata:      metadataCtx,
			Configuration: map[string]any{"project": "TEST", "workflowScheme": "101010"},
		})

		require.NoError(t, err)
		nodeMetadata, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Project)
		require.NotNil(t, nodeMetadata.WorkflowScheme)
		assert.Equal(t, "Support scheme", nodeMetadata.WorkflowScheme.Name)
	})
}

func Test__AssignWorkflowToProject__Execute(t *testing.T) {
	component := AssignWorkflowToProject{}

	t.Run("switches workflow scheme and emits task metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"TEST","name":"Test","style":"classic","simplified":false}`)),
				},
				{
					StatusCode: http.StatusSeeOther,
					Body:       io.NopCloser(strings.NewReader(`{"id":"task-1","status":"ENQUEUED","self":"https://test.atlassian.net/rest/api/3/task/task-1"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":        "TEST",
				"workflowScheme": "101010",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, AssignWorkflowToProjectPayloadType, execCtx.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/rest/api/3/workflowscheme/project/switch")

		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "10000", payload["projectId"])
		assert.Equal(t, "101010", payload["targetSchemeId"])
	})

	t.Run("dry run skips scheme switch", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"TEST","name":"Test","style":"classic","simplified":false}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":        "TEST",
				"workflowScheme": "101010",
				"dryRun":         true,
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		require.Len(t, httpContext.Requests, 1)
	})
}
