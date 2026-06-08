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

func Test__CreateAlert__Setup(t *testing.T) {
	component := CreateAlert{}
	base := jiraTestIntegration()

	t.Run("missing message", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   base,
			Configuration: map[string]any{},
		})
		require.ErrorContains(t, err, "message is required")
	})

	t.Run("valid", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   base,
			Configuration: map[string]any{"message": "CPU high"},
		})
		require.NoError(t, err)
	})
}

func Test__CreateAlert__Execute(t *testing.T) {
	component := CreateAlert{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"result":"Request will be processed","requestId":"rid-1","took":0.1}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"processedAt":"2026-05-01T00:00:00Z","alertId":"new-a1","isSuccess":true}`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"new-a1","message":"CPU high"}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"message": "CPU high",
			"tags":    []any{"prod", "cpu"},
			"responders": []any{
				map[string]any{"id": "team-1", "type": "team"},
			},
		},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.NoError(t, err)
	assert.True(t, execCtx.Finished)
	assert.Equal(t, CreateJiraAlertPayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 3)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/jsm/ops/api/")
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"message":"CPU high"`)
	assert.Contains(t, string(body), `"tags":["prod","cpu"]`)
}

func Test__CreateAlert__Execute__rejects_wrong_alert_message(t *testing.T) {
	component := CreateAlert{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"requestId":"rid-1"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"processedAt":"2026-05-01T00:00:00Z","alertId":"a1","isSuccess":true}`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"a1","message":"old message","alias":"an alias"}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"message": "The earth is flat 5",
			"alias":   "an alias",
		},
		HTTP:           httpContext,
		Integration:    jiraTestIntegration(),
		ExecutionState: execCtx,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "alias")
	require.False(t, execCtx.Finished)
}
