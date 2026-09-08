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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func integrationWithProject() *contexts.IntegrationContext {
	return testIntegration(nil)
}

func projectResponse() *http.Response {
	return jsonResponse(`{"data":{"id":"1","type":"projects","attributes":{"name":"Payments"}}}`)
}

func createdTaskConfiguration() map[string]any {
	return map[string]any{"project": "1", "actions": []string{ActionCreated}}
}

func nodeMetadata(t *testing.T, metadata *contexts.MetadataContext) NodeMetadata {
	t.Helper()

	stored, ok := metadata.Metadata.(NodeMetadata)
	require.True(t, ok, "node metadata must be stored as NodeMetadata")
	return stored
}

// taskWebhookBody builds the JSON:API single-resource body Productive.io
// delivers on a task.created or task.updated webhook.
func taskWebhookBody(id, title string) []byte {
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"id":         id,
			"type":       "tasks",
			"attributes": map[string]any{"title": title},
		},
	})
	return body
}

// signWebhookBody signs body the way SuperPlane verifies it: a hex-encoded
// HMAC-SHA256 keyed with the webhook secret.
func signWebhookBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func webhookHeaders(event, signature string) http.Header {
	headers := http.Header{}
	if event != "" {
		headers.Set(EventHeader, event)
	}
	if signature != "" {
		headers.Set(SignatureHeader, signature)
	}
	return headers
}

func Test__OnTask__Setup(t *testing.T) {
	trigger := &OnTask{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"actions": []string{ActionCreated}},
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
			Configuration: createdTaskConfiguration(),
		})

		require.ErrorContains(t, err, "error finding project")
	})

	t.Run("stores the project and requests a webhook for it", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{projectResponse()}}
		metadata := &contexts.MetadataContext{}
		integration := integrationWithProject()

		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			HTTP:          httpContext,
			Metadata:      metadata,
			Configuration: createdTaskConfiguration(),
		})

		require.NoError(t, err)

		stored := nodeMetadata(t, metadata)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "Payments", stored.Project.Name)

		require.Len(t, integration.WebhookRequests, 1)
		assert.Equal(t, WebhookConfiguration{ProjectID: "1"}, integration.WebhookRequests[0])
	})

	t.Run("does not schedule a poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{projectResponse()}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      requests,
			Configuration: createdTaskConfiguration(),
		})

		require.NoError(t, err)
		assert.Empty(t, requests.Action, "setup must not schedule pollTasks")
	})
}

func Test__OnTask__Hooks__NoHooks(t *testing.T) {
	assert.Empty(t, (&OnTask{}).Hooks(), "the webhook-based trigger defines no hooks")
}

func Test__OnTask__HandleHook__NoOp(t *testing.T) {
	result, err := (&OnTask{}).HandleHook(core.TriggerHookContext{Name: "anything"})
	require.NoError(t, err)
	assert.Nil(t, result)
}

func Test__OnTask__HandleWebhook(t *testing.T) {
	trigger := &OnTask{}

	t.Run("missing event header -> error", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: createdTaskConfiguration(),
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "missing")
	})

	t.Run("unknown event -> ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, body, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("task.deleted", ""),
			Configuration: createdTaskConfiguration(),
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Nil(t, body)
		assert.Zero(t, events.Count())
	})

	t.Run("event not in configured actions -> ignored without checking the signature", func(t *testing.T) {
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionCreated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskUpdatedEvent, ""),
			Configuration: configuration,
			Body:          taskWebhookBody("91", "Fix payment retries"),
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Zero(t, events.Count())
	})

	t.Run("missing signature -> error", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, ""),
			Configuration: createdTaskConfiguration(),
			Body:          taskWebhookBody("91", "Fix payment retries"),
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "missing")
	})

	t.Run("invalid signature -> error", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, "not-the-right-signature"),
			Configuration: createdTaskConfiguration(),
			Body:          taskWebhookBody("91", "Fix payment retries"),
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook signature")
	})

	t.Run("valid task.created delivery -> emits the envelope", func(t *testing.T) {
		body := taskWebhookBody("91", "Fix payment retries")
		events := &contexts.EventContext{}

		code, response, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, signWebhookBody("s3cr3t", body)),
			Configuration: createdTaskConfiguration(),
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Nil(t, response)

		require.Equal(t, 1, events.Count())
		payload := events.Payloads[0]
		assert.Equal(t, TaskPayloadType, payload.Type)

		envelope, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		meta, ok := envelope["meta"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, TaskCreatedEvent, meta["event"])

		document, ok := envelope["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "91", document["id"])
	})

	t.Run("valid task.updated delivery matching updated actions -> emits the envelope", func(t *testing.T) {
		body := taskWebhookBody("91", "Fix payment retries")
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionUpdated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskUpdatedEvent, signWebhookBody("s3cr3t", body)),
			Configuration: configuration,
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
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
