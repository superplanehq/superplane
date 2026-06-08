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

func Test__DeleteAlert__Execute(t *testing.T) {
	component := DeleteAlert{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(strings.NewReader(`{"result":"Request will be processed","requestId":"rid-del","took":0.2}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"alert": "dead-beef"},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, DeleteJiraAlertPayloadType, execCtx.Type)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
}
