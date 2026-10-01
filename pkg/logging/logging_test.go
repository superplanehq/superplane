package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
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

func TestWithWebhookPayload_KeepsJSONAndCutsOversizedBodies(t *testing.T) {
	fields := WithWebhookPayload(log.Fields{"component": ComponentWebhookSentry}, []byte(`{"action":"created"}`))
	raw, ok := fields["payload"].(json.RawMessage)
	require.True(t, ok)
	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, "created", payload["action"])

	oversized := bytes.Repeat([]byte("a"), webhookLogPayloadLimit+1)
	logged := webhookPayloadString(t, WithWebhookPayload(log.Fields{}, oversized))
	assertWebhookPayloadFits(t, logged)

	slashes := bytes.Repeat([]byte{'\\'}, webhookLogPayloadLimit+1)
	assertWebhookPayloadFits(t, webhookPayloadString(t, WithWebhookPayload(log.Fields{}, slashes)))

	controls := bytes.Repeat([]byte{0}, webhookLogPayloadLimit/2)
	assertWebhookPayloadFits(t, webhookPayloadString(t, WithWebhookPayload(log.Fields{}, controls)))

	assert.Nil(t, WithWebhookPayload(nil, nil))
}

func webhookPayloadString(t *testing.T, fields log.Fields) string {
	t.Helper()
	logged, ok := fields["payload"].(string)
	require.True(t, ok)
	return logged
}

func assertWebhookPayloadFits(t *testing.T, logged string) {
	t.Helper()
	assert.True(t, strings.HasSuffix(logged, webhookPayloadTruncated))
	encoded, err := json.Marshal(logged)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), webhookLogPayloadLimit)
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
	assert.Equal(t, "INFO", payloads[0]["severity"])
	assert.Equal(t, "Sentry app webhook received", payloads[0]["message"])
	assert.Equal(t, ComponentWebhookSentry, payloads[0]["component"])
	assert.Equal(t, "issue", payloads[0]["hook_resource"])
	assert.Equal(t, "created", payloads[0]["action"])
	assert.Equal(t, "install-1", payloads[0]["installation_uuid"])
	_, hasLevel := payloads[0]["level"]
	assert.False(t, hasLevel)
	_, hasMsg := payloads[0]["msg"]
	assert.False(t, hasMsg)

	_, processIsText := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, processIsText)
}

func TestLogDatadogWebhookInfo_JSONKeepsTypeAndIntegration(t *testing.T) {
	previousOutput := log.StandardLogger().Out
	previousFormatter := log.StandardLogger().Formatter
	standardOutput := &bytes.Buffer{}
	log.StandardLogger().SetOutput(standardOutput)
	t.Cleanup(func() {
		log.StandardLogger().SetOutput(previousOutput)
		log.StandardLogger().SetFormatter(previousFormatter)
	})

	logger := DatadogWebhookLogger()
	previousLoggerOutput := logger.Out
	logger.SetOutput(standardLogWriter{})
	t.Cleanup(func() {
		logger.SetOutput(previousLoggerOutput)
	})

	LogDatadogWebhookInfo("Datadog webhook received", log.Fields{
		"type":        "event",
		"integration": "sentry",
		"outcome":     "received",
	}, nil)

	payloads := decodeJSONLines(t, standardOutput.String())
	require.Len(t, payloads, 1)
	assert.Equal(t, "info", payloads[0]["level"])
	assert.Equal(t, "Datadog webhook received", payloads[0]["msg"])
	assert.Equal(t, WebhookLogType, payloads[0]["type"])
	assert.Equal(t, DatadogIntegration, payloads[0]["integration"])
	assert.Equal(t, "received", payloads[0]["outcome"])

	_, processIsText := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, processIsText)
	_, webhookIsJSON := logger.Formatter.(*log.JSONFormatter)
	assert.True(t, webhookIsJSON)
}

func TestDatadogWebhookIdentity_ResolvesOnlyWhenAsked(t *testing.T) {
	calls := 0
	logger := WithDatadogWebhookIdentity(log.NewEntry(log.New()), func() log.Fields {
		calls++
		return log.Fields{"workspace_id": "ws-1"}
	})

	assert.Zero(t, calls)
	assert.Equal(t, "ws-1", DatadogWebhookIdentity(logger)["workspace_id"])
	assert.Equal(t, 1, calls)
	assert.Nil(t, DatadogWebhookIdentity(nil))
	assert.Nil(t, DatadogWebhookIdentity(log.NewEntry(log.New())))
	assert.Nil(t, WithDatadogWebhookIdentity(nil, func() log.Fields {
		return log.Fields{}
	}))
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
