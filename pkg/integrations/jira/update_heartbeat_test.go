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

func Test__UpdateHeartbeat__Execute(t *testing.T) {
	component := UpdateHeartbeat{}
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"name":"DNS Checker","interval":10,"intervalUnit":"minutes","enabled":true}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	interval := 10
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"team":      teamID,
			"heartbeat": "DNS Checker",
			"interval":  interval,
		},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, UpdateJiraHeartbeatPayloadType, execCtx.Type)
}

func Test__UpdateHeartbeat__Execute_requires_field(t *testing.T) {
	component := UpdateHeartbeat{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"team":      "team-1",
			"heartbeat": "DNS Checker",
		},
		Integration:    jiraTestIntegration(),
		ExecutionState: &contexts.ExecutionStateContext{},
	})
	require.ErrorContains(t, err, "at least one update field must be enabled")
}

func Test__UpdateHeartbeat__Setup_requires_toggled_field(t *testing.T) {
	component := UpdateHeartbeat{}
	err := component.Setup(core.SetupContext{
		Integration: jiraTestIntegration(),
		Configuration: map[string]any{
			"team":      "team-1",
			"heartbeat": "DNS Checker",
		},
	})
	require.ErrorContains(t, err, "at least one update field must be enabled")
}

func Test__hasAnyHeartbeatUpdate(t *testing.T) {
	assert.False(t, hasAnyHeartbeatUpdate(UpdateHeartbeatSpec{}))
	desc := "x"
	assert.True(t, hasAnyHeartbeatUpdate(UpdateHeartbeatSpec{Description: &desc}))
}
