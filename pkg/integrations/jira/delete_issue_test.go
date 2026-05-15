package jira

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

func Test__DeleteIssue__Setup(t *testing.T) {
	component := DeleteIssue{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegration(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"issueKey": "TEST-1",
			},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing issueKey -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project": "TEST",
			},
		})

		require.ErrorContains(t, err, "issueKey is required")
	})

	t.Run("project not found -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "OTHER", Name: "Other"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
		})

		require.ErrorContains(t, err, "project TEST not found")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "TEST", Name: "Test"}},
			}),
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteIssue__Execute(t *testing.T) {
	component := DeleteIssue{}

	t.Run("successful delete", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1"}`))},
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(``))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, DeleteIssuePayloadType, execCtx.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	})

	t.Run("deleteSubtasks flag is forwarded", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1"}`))},
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(``))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":        "TEST",
				"issueKey":       "TEST-1",
				"deleteSubtasks": true,
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "deleteSubtasks=true")
	})

	t.Run("get failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["nope"]}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch issue before delete")
	})

	t.Run("delete failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-1"}`))},
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["no permission"]}`))},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "TEST",
				"issueKey": "TEST-1",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete issue")
	})
}

func Test__DeleteIssue__ComponentInfo(t *testing.T) {
	component := DeleteIssue{}
	assert.Equal(t, "jira.deleteIssue", component.Name())
	assert.Equal(t, "Delete Issue", component.Label())
	assert.NotEmpty(t, component.Documentation())
}
