package onvmcreate

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_OnVMCreated_Metadata(t *testing.T) {
	trigger := &OnVMCreated{}
	assert.Equal(t, "gcp.onVMCreated", trigger.Name())
	assert.Equal(t, "On VM Created", trigger.Label())
	assert.Equal(t, "Emits when a new Compute Engine VM is created (provisioning succeeded). Trigger uses a Cloud Logging sink to Pub/Sub and emits the VM creation payload to start SuperPlane workflow executions.", trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
	assert.Nil(t, trigger.Actions())
}

func Test_OnVMCreated_Configuration(t *testing.T) {
	trigger := &OnVMCreated{}
	fields := trigger.Configuration()
	require.Len(t, fields, 2)
	assert.Equal(t, "projectId", fields[0].Name)
	assert.Equal(t, "Project ID", fields[0].Label)
	assert.True(t, fields[0].Required)
	assert.Equal(t, "region", fields[1].Name)
	assert.Equal(t, "Region", fields[1].Label)
	assert.Equal(t, "us-central1", fields[1].Default)
}

func Test_OnVMCreated_ExampleData(t *testing.T) {
	trigger := &OnVMCreated{}
	data := trigger.ExampleData()
	assert.Equal(t, auditLogEventType, data["type"])
	assert.Equal(t, computeServiceName, data["serviceName"])
	assert.Equal(t, instancesInsertMethod, data["methodName"])
	assert.Equal(t, "projects/my-project/zones/us-central1-a/instances/my-vm", data["resourceName"])
}

func Test_resolvePayloadBytes(t *testing.T) {
	t.Run("raw JSON returns body as-is", func(t *testing.T) {
		body := []byte(`{"type":"some-event"}`)
		out, err := resolvePayloadBytes(body)
		require.NoError(t, err)
		assert.Equal(t, body, out)
	})

	t.Run("non-PubSub JSON returns body as-is", func(t *testing.T) {
		body := []byte(`{"not":"a pubsub envelope"}`)
		out, err := resolvePayloadBytes(body)
		require.NoError(t, err)
		assert.Equal(t, body, out)
	})

	t.Run("PubSub envelope with empty message.data returns body as-is", func(t *testing.T) {
		body := []byte(`{"message":{},"subscription":"sub"}`)
		out, err := resolvePayloadBytes(body)
		require.NoError(t, err)
		assert.Equal(t, body, out)
	})

	t.Run("PubSub envelope with base64 message.data decodes", func(t *testing.T) {
		inner := []byte(`{"type":"cloud.event"}`)
		encoded := base64.StdEncoding.EncodeToString(inner)
		body, _ := json.Marshal(map[string]any{
			"message":      map[string]any{"data": encoded},
			"subscription": "sub",
		})
		out, err := resolvePayloadBytes(body)
		require.NoError(t, err)
		assert.Equal(t, inner, out)
	})

	t.Run("invalid base64 in message.data returns error", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"message":      map[string]any{"data": "not-valid-base64!!"},
			"subscription": "sub",
		})
		out, err := resolvePayloadBytes(body)
		require.Error(t, err)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), "base64")
	})
}

func Test_isCompletionEvent(t *testing.T) {
	assert.True(t, isCompletionEvent(nil))
	assert.True(t, isCompletionEvent(&auditLogOperation{Last: true}))
	assert.False(t, isCompletionEvent(&auditLogOperation{Last: false}))
}

func Test_operationFromData(t *testing.T) {
	assert.Nil(t, operationFromData(nil))
	assert.Nil(t, operationFromData(map[string]any{}))
	assert.Nil(t, operationFromData(map[string]any{"operation": "not-a-map"}))
	op := operationFromData(map[string]any{"operation": map[string]any{"last": true}})
	require.NotNil(t, op)
	assert.True(t, op.Last)
	op2 := operationFromData(map[string]any{"operation": map[string]any{"last": false}})
	require.NotNil(t, op2)
	assert.False(t, op2.Last)
}

func Test_normalizedFromEnvelope(t *testing.T) {
	payload := &EventPayload{
		Type:         auditLogEventType,
		Source:       "//cloudaudit.googleapis.com/...",
		ServiceName:  computeServiceName,
		MethodName:   "  " + instancesInsertMethod + "  ",
		ResourceName: "projects/p/zones/z/instances/vm1",
		Data:         map[string]any{"operation": map[string]any{"last": true}},
	}
	svc, method, resource, eventData, op := normalizedFromEnvelope(payload)
	assert.Equal(t, computeServiceName, svc)
	assert.Equal(t, instancesInsertMethod, method)
	assert.Equal(t, "projects/p/zones/z/instances/vm1", resource)
	assert.NotNil(t, eventData)
	assert.Equal(t, instancesInsertMethod, eventData["methodName"])
	assert.NotNil(t, op)
	assert.True(t, op.Last)
}

func Test_OnVMCreated_HandleWebhook(t *testing.T) {
	trigger := &OnVMCreated{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not json"),
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        &contexts.EventContext{},
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("valid JSON but not audit event returns 200 no emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"type":"other-event"}`),
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("CloudEvents envelope wrong service returns 200 no emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  "storage.googleapis.com",
			"methodName":   instancesInsertMethod,
			"resourceName": "projects/p/zones/z/instances/vm1",
			"data":         map[string]any{"operation": map[string]any{"last": true}},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("CloudEvents envelope wrong method returns 200 no emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   "v1.compute.instances.delete",
			"resourceName": "projects/p/zones/z/instances/vm1",
			"data":         map[string]any{"operation": map[string]any{"last": true}},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("CloudEvents envelope operation not completion returns 200 no emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethod,
			"resourceName": "projects/p/zones/z/instances/vm1",
			"data":         map[string]any{"operation": map[string]any{"last": false}},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("CloudEvents envelope VM insert completion emits event", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"source":       "//cloudaudit.googleapis.com/...",
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethod,
			"resourceName": "projects/my-proj/zones/us-central1-a/instances/my-vm",
			"data":         map[string]any{"operation": map[string]any{"last": true}},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
		assert.Equal(t, instancesInsertMethod, events.Payloads[0].Data.(map[string]any)["methodName"])
	})

	t.Run("project filter mismatch returns 200 no emit", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethod,
			"resourceName": "projects/other-proj/zones/us-central1-a/instances/vm1",
			"data":         map[string]any{"operation": map[string]any{"last": true}},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{"projectId": "my-proj"},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("project filter match emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethod,
			"resourceName": "projects/my-proj/zones/us-central1-a/instances/vm1",
			"data":         map[string]any{"operation": map[string]any{"last": true}},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{"projectId": "my-proj"},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("log entry format VM insert completion emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"protoPayload": map[string]any{
				"serviceName":  computeServiceName,
				"methodName":   instancesInsertMethod,
				"resourceName": "projects/p/zones/z/instances/vm1",
			},
			"logName":   "projects/p/logs/activity",
			"timestamp": "2025-02-14T12:00:00Z",
			"insertId":  "id1",
			"operation": map[string]any{"last": true},
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("PubSub push envelope decodes and processes", func(t *testing.T) {
		events := &contexts.EventContext{}
		inner := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethod,
			"resourceName": "projects/p/zones/z/instances/vm1",
			"data":         map[string]any{"operation": map[string]any{"last": true}},
		})
		envelope := mustJSON(t, map[string]any{
			"message":      map[string]any{"data": base64.StdEncoding.EncodeToString(inner)},
			"subscription": "projects/p/subscriptions/sub",
		})
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          envelope,
			Configuration: map[string]any{},
			Logger:        logger,
			Events:        events,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
