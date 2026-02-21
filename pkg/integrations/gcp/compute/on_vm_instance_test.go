package compute

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_OnVMInstance_Metadata(t *testing.T) {
	trigger := &OnVMInstance{}
	assert.Equal(t, "gcp.compute.onVMInstance", trigger.Name())
	assert.Equal(t, "Compute • On VM Instance", trigger.Label())
	assert.Equal(t, "Listen to GCP Compute Engine VM instance lifecycle events", trigger.Description())
	assert.NotEmpty(t, trigger.Documentation())
	assert.Equal(t, "gcp", trigger.Icon())
	assert.Equal(t, "gray", trigger.Color())
	assert.Nil(t, trigger.Actions())
}

func Test_OnVMInstance_Configuration(t *testing.T) {
	trigger := &OnVMInstance{}
	fields := trigger.Configuration()
	require.Len(t, fields, 2)
	assert.Equal(t, "projectId", fields[0].Name)
	assert.Equal(t, "Project ID", fields[0].Label)
	assert.True(t, fields[0].Required)
	assert.Equal(t, "region", fields[1].Name)
	assert.Equal(t, "Region", fields[1].Label)
	assert.Equal(t, "us-central1", fields[1].Default)
}

func Test_OnVMInstance_ExampleData(t *testing.T) {
	trigger := &OnVMInstance{}
	data := trigger.ExampleData()
	assert.Equal(t, auditLogEventType, data["type"])
	assert.Equal(t, computeServiceName, data["serviceName"])
	assert.Equal(t, instancesInsertMethod, data["methodName"])
	assert.Equal(t, "projects/my-project/zones/us-central1-a/instances/my-vm", data["resourceName"])
}

func Test_OnVMInstance_InstanceCelFilter(t *testing.T) {
	assert.Contains(t, InstanceCelFilter, "google.cloud.audit.log.v1.written")
	assert.Contains(t, InstanceCelFilter, "compute.googleapis.com")
	assert.Contains(t, InstanceCelFilter, "v1.compute.instances.insert")
	assert.Contains(t, InstanceCelFilter, "beta.compute.instances.insert")
	assert.Contains(t, InstanceCelFilter, "compute.instances.insert")
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

func Test_OnVMInstance_HandleWebhook(t *testing.T) {
	trigger := &OnVMInstance{}
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

	t.Run("beta method name emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethodBeta,
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
		require.Equal(t, 1, events.Count())
		assert.Equal(t, EmittedEventType, events.Payloads[0].Type)
	})

	t.Run("short method name emits", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := mustJSON(t, map[string]any{
			"type":         auditLogEventType,
			"serviceName":  computeServiceName,
			"methodName":   instancesInsertMethodShort,
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
