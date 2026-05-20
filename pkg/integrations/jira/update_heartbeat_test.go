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

func Test__UpdateHeartbeat__Setup_rejects_empty_toggled_field(t *testing.T) {
	component := UpdateHeartbeat{}
	empty := ""
	err := component.Setup(core.SetupContext{
		Integration: jiraTestIntegration(),
		Configuration: map[string]any{
			"team":        "team-1",
			"heartbeat":   "DNS Checker",
			"description": empty,
		},
	})
	require.ErrorContains(t, err, "at least one update field must be enabled")
}

func Test__hasEffectiveHeartbeatUpdate(t *testing.T) {
	assert.False(t, hasEffectiveHeartbeatUpdate(UpdateHeartbeatSpec{}))
	empty := ""
	assert.False(t, hasEffectiveHeartbeatUpdate(UpdateHeartbeatSpec{Description: &empty}))
	desc := "x"
	assert.True(t, hasEffectiveHeartbeatUpdate(UpdateHeartbeatSpec{Description: &desc}))
	interval := 5
	assert.True(t, hasEffectiveHeartbeatUpdate(UpdateHeartbeatSpec{Interval: &interval}))

	// intervalUnit alone (interval toggle off) must not be treated as a valid update.
	unit := "minutes"
	assert.False(t, hasEffectiveHeartbeatUpdate(UpdateHeartbeatSpec{IntervalUnit: &unit}))
}

func Test__updateHeartbeatRequestFromSpec_intervalUnit(t *testing.T) {
	unit := "hours"
	interval := 2

	t.Run("intervalUnit is omitted when interval is absent", func(t *testing.T) {
		req := updateHeartbeatRequestFromSpec(UpdateHeartbeatSpec{IntervalUnit: &unit})
		assert.Nil(t, req.Interval)
		assert.Empty(t, req.IntervalUnit)
	})

	t.Run("intervalUnit is included when interval is present", func(t *testing.T) {
		req := updateHeartbeatRequestFromSpec(UpdateHeartbeatSpec{Interval: &interval, IntervalUnit: &unit})
		assert.Equal(t, &interval, req.Interval)
		assert.Equal(t, "hours", req.IntervalUnit)
	})

	t.Run("interval without intervalUnit is still valid", func(t *testing.T) {
		req := updateHeartbeatRequestFromSpec(UpdateHeartbeatSpec{Interval: &interval})
		assert.Equal(t, &interval, req.Interval)
		assert.Empty(t, req.IntervalUnit)
	})
}
