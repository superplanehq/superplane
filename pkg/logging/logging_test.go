package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductiveWebhookFailure_JSONIncludesComponentAndWebhookType(t *testing.T) {
	previousOutput := log.StandardLogger().Out
	previousFormatter := log.StandardLogger().Formatter
	standardOutput := &bytes.Buffer{}
	log.StandardLogger().SetOutput(standardOutput)
	t.Cleanup(func() {
		log.StandardLogger().SetOutput(previousOutput)
		log.StandardLogger().SetFormatter(previousFormatter)
	})

	logger := ProductiveWebhookLogger()
	previousLoggerOutput := logger.Out
	logger.SetOutput(standardLogWriter{})
	t.Cleanup(func() {
		logger.SetOutput(previousLoggerOutput)
	})

	LogProductiveWebhookFailure("task.updated", log.Fields{
		"status": 500,
	}, errors.New("productive task 123: task update activity unavailable"))
	LogProductiveWebhookFailure("task.created", nil, errors.New("productive task 91: task list unavailable"))
	LogProductiveWebhookFailure(" ", nil, errors.New("event missing"))

	payloads := decodeJSONLines(t, standardOutput.String())
	require.Len(t, payloads, 3)

	assert.Equal(t, ComponentWebhookProductive, payloads[0]["component"])
	assert.Equal(t, "task.updated", payloads[0]["webhookType"])
	assert.Equal(t, "error", payloads[0]["level"])
	assert.Equal(t, "error handling webhook", payloads[0]["msg"])
	assert.EqualValues(t, 500, payloads[0]["status"])
	assert.Contains(t, payloads[0]["error"], "task update activity unavailable")

	assert.Equal(t, ComponentWebhookProductive, payloads[1]["component"])
	assert.Equal(t, "task.created", payloads[1]["webhookType"])

	assert.Equal(t, ComponentWebhookProductive, payloads[2]["component"])
	assert.Equal(t, "unknown", payloads[2]["webhookType"])

	_, processIsText := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, processIsText)
	_, webhookIsJSON := logger.Formatter.(*log.JSONFormatter)
	assert.True(t, webhookIsJSON)
}

func TestProductiveWebhookWarning_JSONUsesWarningLevel(t *testing.T) {
	logger := ProductiveWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})

	LogProductiveWebhookWarning("task.updated", "task update emitted without task list move", log.Fields{
		"component": "overridden",
	}, errors.New("no activity"))

	payloads := decodeJSONLines(t, buffer.String())
	require.Len(t, payloads, 1)
	assert.Equal(t, "warning", payloads[0]["level"])
	assert.Equal(t, "task update emitted without task list move", payloads[0]["msg"])
	assert.Equal(t, ComponentWebhookProductive, payloads[0]["component"])
	assert.Equal(t, "task.updated", payloads[0]["webhookType"])
	assert.Equal(t, "no activity", payloads[0]["error"])
}

func TestLogSentryWebhookInfo_JSONIncludesComponent(t *testing.T) {
	logger := SentryWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})

	LogSentryWebhookInfo("Sentry app webhook received", log.Fields{
		"hook_resource":     "issue",
		"action":            "created",
		"installation_uuid": "install-1",
	})

	payloads := decodeJSONLines(t, buffer.String())
	require.Len(t, payloads, 1)
	assert.Equal(t, "info", payloads[0]["level"])
	assert.Equal(t, "Sentry app webhook received", payloads[0]["msg"])
	assert.Equal(t, ComponentWebhookSentry, payloads[0]["component"])
	assert.Equal(t, "issue", payloads[0]["hook_resource"])
	assert.Equal(t, "created", payloads[0]["action"])
	assert.Equal(t, "install-1", payloads[0]["installation_uuid"])

	_, processIsText := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, processIsText)
}

func TestWithWebhookNode_AddsOrganizationCanvasAndWebhook(t *testing.T) {
	entry := WithWebhookNode(log.NewEntry(log.New()), WebhookNodeFields{
		OrganizationID: "org-1",
		CanvasID:       "canvas-1",
		WebhookID:      "webhook-1",
	})

	assert.Equal(t, "org-1", entry.Data["organization_id"])
	assert.Equal(t, "canvas-1", entry.Data["workflow_id"])
	assert.Equal(t, "webhook-1", entry.Data["webhook_id"])

	withoutOrganization := WithWebhookNode(log.NewEntry(log.New()), WebhookNodeFields{
		CanvasID: "canvas-1",
	})
	_, hasOrganization := withoutOrganization.Data["organization_id"]
	assert.False(t, hasOrganization)
	_, hasWebhook := withoutOrganization.Data["webhook_id"]
	assert.False(t, hasWebhook)
	assert.Equal(t, "canvas-1", withoutOrganization.Data["workflow_id"])
}

func decodeJSONLines(t *testing.T, raw string) []map[string]any {
	t.Helper()

	payloads := []map[string]any{}
	for _, line := range bytes.Split([]byte(raw), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &payload))
		payloads = append(payloads, payload)
	}
	return payloads
}
