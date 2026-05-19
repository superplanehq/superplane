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
			Configuration: map[string]any{"alert": "x"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one update")
	})

	t.Run("patch existing note missing body", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: base,
			Configuration: map[string]any{
				"alert": "x",
				"patchExistingNote": map[string]any{
					"noteId": "n1",
					"note":   "",
				},
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "requires both Note ID and Note text")
	})

	t.Run("valid acknowledge only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: base,
			Configuration: map[string]any{
				"alert":            "x",
				"acknowledgeAlert": true,
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("assign without assignee", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: base,
			Configuration: map[string]any{
				"alert":         "x",
				"setAssignment": true,
				"assignee":      "",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "assignee is required")
	})

	t.Run("new note keyed as null does not block other operations", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: base,
			Configuration: map[string]any{
				"alert":            "x",
				"newNote":          nil,
				"acknowledgeAlert": true,
			},
			Metadata: &contexts.MetadataContext{},
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
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"processedAt":"2026-05-01T00:00:00Z","alertId":"","isSuccess":true}`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"alert-1","message":"after"}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Integration:    jiraTestIntegration(),
		HTTP:           httpContext,
		ExecutionState: execCtx,
		Configuration: map[string]any{
			"alert":            "alert-1",
			"setDescription":   true,
			"description":      "desc",
			"setMessage":       true,
			"message":          "msg",
			"setPriority":      true,
			"priority":         "P2",
			"newNote":          "hello",
			"acknowledgeAlert": true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, UpdateJiraAlertPayloadType, execCtx.Type)
	require.Len(t, httpContext.Requests, 7)
	assert.Equal(t, http.MethodPatch, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/description")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "/message")
	assert.Contains(t, httpContext.Requests[2].URL.String(), "/priority")
	assert.Contains(t, httpContext.Requests[3].URL.String(), "/notes")
	assert.Contains(t, httpContext.Requests[4].URL.String(), "/acknowledge")
	assert.Contains(t, httpContext.Requests[5].URL.String(), "/requests/")
	assert.Contains(t, httpContext.Requests[6].URL.String(), "/alert-1")

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

func Test__validateUpdateAlertConfigurable__priorityWhenEnabled(t *testing.T) {
	t.Run("with other operations", func(t *testing.T) {
		cfg := map[string]any{
			"setPriority": true,
			"setMessage":  true,
		}
		err := validateUpdateAlertConfigurable(cfg, UpdateAlertSpec{Alert: "x", Message: "m", Priority: "__none__"})
		require.ErrorContains(t, err, `choose a concrete priority`)

		require.NoError(t, validateUpdateAlertConfigurable(cfg, UpdateAlertSpec{Alert: "x", Message: "m", Priority: "P3"}))
	})

	t.Run("priority only with default __none__", func(t *testing.T) {
		cfg := map[string]any{"setPriority": true}
		err := validateUpdateAlertConfigurable(cfg, UpdateAlertSpec{Alert: "x", Priority: "__none__"})
		require.ErrorContains(t, err, `choose a concrete priority`)
		require.NotContains(t, err.Error(), "enable at least one update")
	})
}
