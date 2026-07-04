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

func Test__GetIncident__Execute(t *testing.T) {
	component := GetIncident{}

	t.Run("by numeric id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"summary":"Sev1"}`)),
				},
			},
		}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"issue": "10050"},
			HTTP:           httpContext,
			Integration:    jiraTestIntegration(),
			ExecutionState: execCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, GetJiraIncidentPayloadType, execCtx.Type)
	})

	t.Run("by issue key resolves then gets incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10050","key":"ITSM-30","self":"x","fields":{}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"summary":"via key"}`)),
				},
			},
		}
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"issue": "ITSM-30"},
			HTTP:           httpContext,
			Integration:    jiraTestIntegration(),
			ExecutionState: execCtx,
		})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
	})
}
