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

func Test__ApproveWorkflow__Setup(t *testing.T) {
	component := ApproveWorkflow{}

	t.Run("missing issue key -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"decision": "approve"},
		})

		require.ErrorContains(t, err, "issueKey is required")
	})

	t.Run("invalid decision -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"issueKey": "ITSM-1", "decision": "hold"},
		})

		require.ErrorContains(t, err, "decision must be approve or decline")
	})

	t.Run("approval id is required when selector is byId", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueKey":         "ITSM-1",
				"decision":         "approve",
				"approvalSelector": approvalSelectorByID,
			},
		})

		require.ErrorContains(t, err, "approvalId is required")
	})
}

func Test__ApproveWorkflow__Execute(t *testing.T) {
	component := ApproveWorkflow{}

	t.Run("approves latest pending approval", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"issueKey":"ITSM-1","serviceDeskId":"1","requestTypeId":"10"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","name":"Old","finalDecision":"approved"},{"id":"2","name":"Manager","finalDecision":"PENDING"}],"isLastPage":true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"2","name":"Manager","finalDecision":"approved"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"issueKey":         "ITSM-1",
				"decision":         "approve",
				"approvalSelector": approvalSelectorLatestPending,
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ApproveWorkflowPayloadType, execCtx.Type)
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/rest/servicedeskapi/request/ITSM-1/approval/2")

		body, err := io.ReadAll(httpContext.Requests[2].Body)
		require.NoError(t, err)
		var payload map[string]string
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "approve", payload["decision"])
	})

	t.Run("no pending approval -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"issueKey":"ITSM-1","serviceDeskId":"1"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","finalDecision":"approved"}],"isLastPage":true}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"issueKey":         "ITSM-1",
				"decision":         "approve",
				"approvalSelector": approvalSelectorLatestPending,
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "no pending approval")
	})

	t.Run("permission failure explains approver requirement", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"issueKey":"ITSM-1","serviceDeskId":"1"}`)),
				},
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessage":"forbidden"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"issueKey":         "ITSM-1",
				"decision":         "decline",
				"approvalSelector": approvalSelectorByID,
				"approvalId":       "2",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "approver list")
	})

	t.Run("standard Jira issue is rejected", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessage":"not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"issueKey":         "PROJ-1",
				"decision":         "approve",
				"approvalSelector": approvalSelectorByID,
				"approvalId":       "2",
			},
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Jira Service Management request")
	})
}
