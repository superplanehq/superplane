package productive

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testWebhookSecret = "webhook-secret"

func integrationWithProject() *contexts.IntegrationContext {
	return testIntegration(nil)
}

func signedTaskRequest(t *testing.T, eventType string, body map[string]any, config map[string]any, events *contexts.EventContext) core.WebhookRequestContext {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(raw)

	headers := http.Header{}
	headers.Set(EventHeader, eventType)
	headers.Set(SignatureHeader, hex.EncodeToString(mac.Sum(nil)))

	return core.WebhookRequestContext{
		Headers:       headers,
		Body:          raw,
		Configuration: config,
		Webhook:       &contexts.NodeWebhookContext{Secret: testWebhookSecret},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	}
}

func taskEvent() map[string]any {
	return map[string]any{
		"meta": map[string]any{"event": TaskCreatedEvent},
		"data": map[string]any{
			"id":   "1",
			"type": "tasks",
			"attributes": map[string]any{
				"title": "Fix payment retries",
			},
		},
	}
}

func Test__OnTask__Setup(t *testing.T) {
	trigger := &OnTask{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"actions": []string{"created"}},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("missing actions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			HTTP:          &contexts.HTTPContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "1"},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	// The shared multi-select validation lets an empty list satisfy Required,
	// so Setup has to reject it or the trigger would never match anything.
	t.Run("empty actions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			HTTP:          &contexts.HTTPContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "1", "actions": []string{}},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	t.Run("unknown project -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"errors":[{"title":"Not found"}]}`))},
		}}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			HTTP:          httpContext,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": "1", "actions": []string{"created"}},
		})

		require.ErrorContains(t, err, "error finding project")
	})

	t.Run("requests a webhook scoped to the project", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(`{"data":{"id":"1","type":"projects","attributes":{"name":"Payments"}}}`),
		}}
		metadataContext := &contexts.MetadataContext{}
		integration := integrationWithProject()

		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			HTTP:          httpContext,
			Metadata:      metadataContext,
			Configuration: map[string]any{"project": "1", "actions": []string{"created"}},
		})

		require.NoError(t, err)

		metadata, ok := metadataContext.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Project)
		assert.Equal(t, "Payments", metadata.Project.Name)

		require.Len(t, integration.WebhookRequests, 1)
		webhookConfig, ok := integration.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "1", webhookConfig.ProjectID)
		assert.Equal(t, []string{TaskCreatedEvent}, webhookConfig.Events)
	})
}

func Test__OnTask__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnTask{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "1", "actions": []string{"created"}},
	})

	assert.Equal(t, http.StatusBadRequest, code)
	require.ErrorContains(t, err, EventHeader)
}

func Test__OnTask__HandleWebhook__IgnoresOtherEvents(t *testing.T) {
	trigger := &OnTask{}
	events := &contexts.EventContext{}

	headers := http.Header{}
	headers.Set(EventHeader, "project.updated")

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "1", "actions": []string{"created"}},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnTask__HandleWebhook__InvalidSignature(t *testing.T) {
	trigger := &OnTask{}
	events := &contexts.EventContext{}

	ctx := signedTaskRequest(t, TaskCreatedEvent, taskEvent(), map[string]any{"project": "1", "actions": []string{"created"}}, events)
	ctx.Headers.Set(SignatureHeader, "deadbeef")

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, "invalid webhook signature")
	assert.Zero(t, events.Count())
}

func Test__OnTask__HandleWebhook__MissingSignature(t *testing.T) {
	trigger := &OnTask{}
	events := &contexts.EventContext{}

	ctx := signedTaskRequest(t, TaskCreatedEvent, taskEvent(), map[string]any{"project": "1", "actions": []string{"created"}}, events)
	ctx.Headers.Del(SignatureHeader)

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, SignatureHeader)
}

func Test__OnTask__HandleWebhook__Success(t *testing.T) {
	trigger := &OnTask{}
	events := &contexts.EventContext{}

	ctx := signedTaskRequest(t, TaskCreatedEvent, taskEvent(), map[string]any{"project": "1", "actions": []string{"created"}}, events)

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)

	require.Equal(t, 1, events.Count())
	assert.Equal(t, TaskPayloadType, events.Payloads[0].Type)

	data, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	envelope := data["data"].(map[string]any)
	attributes := envelope["attributes"].(map[string]any)
	assert.Equal(t, "Fix payment retries", attributes["title"])
}

func Test__OnTask__HandleWebhook__FiltersActions(t *testing.T) {
	trigger := &OnTask{}

	t.Run("action not selected is ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedTaskRequest(t, TaskUpdatedEvent, taskEvent(), map[string]any{"project": "1", "actions": []string{"created"}}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	// Regression: an empty list must match nothing rather than falling through
	// and emitting every action.
	t.Run("empty actions emits nothing", func(t *testing.T) {
		for _, eventType := range []string{TaskCreatedEvent, TaskUpdatedEvent} {
			events := &contexts.EventContext{}
			ctx := signedTaskRequest(t, eventType, taskEvent(), map[string]any{"project": "1", "actions": []string{}}, events)

			code, _, err := trigger.HandleWebhook(ctx)
			assert.Equal(t, http.StatusOK, code)
			require.NoError(t, err)
			assert.Zero(t, events.Count(), "event %q must not be emitted with an empty action list", eventType)
		}
	})

	t.Run("updated action is delivered when selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedTaskRequest(t, TaskUpdatedEvent, taskEvent(), map[string]any{"project": "1", "actions": []string{"created", "updated"}}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__OnTask__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnTask{}
	example := trigger.ExampleData()

	assert.Equal(t, TaskPayloadType, example["type"])
	require.NotEmpty(t, example["timestamp"])

	envelope, ok := example["data"].(map[string]any)
	require.True(t, ok)

	meta, ok := envelope["meta"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, TaskCreatedEvent, meta["event"])

	task, ok := envelope["data"].(map[string]any)
	require.True(t, ok)
	attributes, ok := task["attributes"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, attributes["title"])
}
