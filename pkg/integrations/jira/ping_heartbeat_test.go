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

func Test__PingHeartbeat__Execute(t *testing.T) {
	component := PingHeartbeat{}
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(strings.NewReader(`{"message":"PONG - Heartbeat received"}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"team":      teamID,
			"heartbeat": "DNS Checker",
		},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, PingJiraHeartbeatPayloadType, execCtx.Type)
}
