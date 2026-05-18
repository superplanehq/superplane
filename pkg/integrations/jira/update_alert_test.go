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

func Test__UpdateAlert__Setup(t *testing.T) {
	component := UpdateAlert{}
	base := jiraTestIntegration()

	t.Run("no operations", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   base,
			Configuration: map[string]any{"alertId": "x"},
		})
		require.ErrorContains(t, err, "at least one update")
	})

	t.Run("existing note id without body", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   base,
			Configuration: map[string]any{"alertId": "x", "existingNoteId": "n1"},
		})
		require.ErrorContains(t, err, "existingNote is required")
	})

	t.Run("valid acknowledge only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: base,
			Configuration: map[string]any{
				"alertId":          "x",
				"acknowledgeAlert": true,
			},
		})
		require.NoError(t, err)
	})
}

func Test__UpdateAlert__Execute__combined(t *testing.T) {
	component := UpdateAlert{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{"requestId":"p1"}`))},
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{"requestId":"p2"}`))},
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{"requestId":"p3"}`))},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"note-new","note":"hello"}`)),
			},
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(`{"requestId":"ack"}`))},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Integration:    jiraTestIntegration(),
		HTTP:           httpContext,
		ExecutionState: execCtx,
		Configuration: map[string]any{
			"alertId":          "alert-1",
			"description":      "desc",
			"message":          "msg",
			"newNote":          "hello",
			"acknowledgeAlert": true,
			"priority":         "P2",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, UpdateJiraAlertPayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 5)
	assert.Equal(t, http.MethodPatch, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/description")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "/message")
	assert.Contains(t, httpContext.Requests[2].URL.String(), "/priority")
	assert.Contains(t, httpContext.Requests[3].URL.String(), "/notes")
	assert.Contains(t, httpContext.Requests[4].URL.String(), "/acknowledge")

	descBody, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(descBody), "desc")
	msgBody, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)
	assert.Contains(t, string(msgBody), "msg")
	prBody, err := io.ReadAll(httpContext.Requests[2].Body)
	require.NoError(t, err)
	assert.Contains(t, string(prBody), "P2")
}

func Test__validateUpdateAlertHasOperations__priorityNone(t *testing.T) {
	err := validateUpdateAlertHasOperations(UpdateAlertSpec{AlertID: "x", Priority: "__none__"})
	require.Error(t, err)
	require.NoError(t, validateUpdateAlertHasOperations(UpdateAlertSpec{AlertID: "x", Priority: "P3"}))
}
