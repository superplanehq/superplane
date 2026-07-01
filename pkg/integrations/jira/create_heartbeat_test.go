package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateHeartbeat__Setup(t *testing.T) {
	component := CreateHeartbeat{}

	t.Run("missing team", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   jiraTestIntegration(),
			Configuration: map[string]any{"name": "Checker", "interval": 5, "intervalUnit": "minutes"},
		})
		require.ErrorContains(t, err, "team is required")
	})

	t.Run("missing name", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   jiraTestIntegration(),
			Configuration: map[string]any{"team": "team-1", "interval": 5, "intervalUnit": "minutes"},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"platformTeams":[{"teamId":"` + teamID + `","teamName":"Ops"}]}`)),
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP:          httpContext,
			Integration:   jiraTestIntegration(),
			Metadata:      metadataCtx,
			Configuration: map[string]any{"team": teamID, "name": "Checker", "interval": 5, "intervalUnit": "minutes"},
		})
		require.NoError(t, err)
		var meta CreateHeartbeatNodeMetadata
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &meta))
		assert.Equal(t, "Ops", meta.TeamName)
	})
}

func Test__CreateHeartbeat__Execute(t *testing.T) {
	component := CreateHeartbeat{}
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"name":"Checker","interval":5,"intervalUnit":"minutes","enabled":true}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"team":         teamID,
			"name":         "Checker",
			"interval":     5,
			"intervalUnit": "minutes",
			"enabled":      true,
		},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, CreateJiraHeartbeatPayloadType, execCtx.Type)
}

func Test__CreateHeartbeat__Execute_defaults_enabled_true(t *testing.T) {
	component := CreateHeartbeat{}
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"name":"Checker","interval":5,"intervalUnit":"minutes","enabled":true}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"team":         teamID,
			"name":         "Checker",
			"interval":     5,
			"intervalUnit": "minutes",
		},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, CreateJiraHeartbeatPayloadType, execCtx.Type)
}

func Test__createHeartbeatEnabledFromSpec(t *testing.T) {
	assert.True(t, createHeartbeatEnabledFromSpec(CreateHeartbeatSpec{}))
	disabled := false
	assert.False(t, createHeartbeatEnabledFromSpec(CreateHeartbeatSpec{Enabled: &disabled}))
	enabled := true
	assert.True(t, createHeartbeatEnabledFromSpec(CreateHeartbeatSpec{Enabled: &enabled}))
}
