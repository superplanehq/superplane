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

func Test__GetAlert__Setup(t *testing.T) {
	component := GetAlert{}
	require.ErrorContains(t, component.Setup(core.SetupContext{
		Integration:   jiraTestIntegration(),
		Configuration: map[string]any{},
	}), "alert is required")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"e0caa0ce-d52f-4500-81b9-d592d06970b6","message":"CPU","tinyId":"42"}`)),
			},
		},
	}
	meta := &contexts.MetadataContext{}
	require.NoError(t, component.Setup(core.SetupContext{
		Integration:   jiraTestIntegration(),
		Configuration: map[string]any{"alert": " e0caa0ce-d52f-4500-81b9-d592d06970b6 "},
		HTTP:          httpContext,
		Metadata:      meta,
	}))
	picker, ok := meta.Metadata.(OpsAlertPickerMetadata)
	require.True(t, ok)
	require.NotEmpty(t, picker.AlertLabel)
	require.Contains(t, picker.AlertLabel, "CPU")
}

func Test__GetAlert__Execute(t *testing.T) {
	component := GetAlert{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"e0caa0ce-d52f-4500-81b9-d592d06970b6","message":"CPU","status":"open"}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"alert": "e0caa0ce-d52f-4500-81b9-d592d06970b6"},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, GetJiraAlertPayloadType, execCtx.Type)
}
