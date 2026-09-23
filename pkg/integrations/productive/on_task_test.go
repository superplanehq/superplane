package productive

import (
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
func taskWebhookBody(id, projectID, title string) []byte {
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"id":   id,
			"type": "tasks",
			"attributes": map[string]any{
				"title": title,
			},
			"relationships": map[string]any{
				"project": map[string]any{
					"data": map[string]any{"type": "projects", "id": projectID},
				},
				"assignee": map[string]any{"data": nil},
			},
		},
		"included": []any{
			map[string]any{
				"id":         projectID,
				"type":       "projects",
				"attributes": map[string]any{"name": "sentry-intake-test-project"},
			},
		},
	})
	return body
}

// signWebhookBody signs body the way Productive.io does: HMAC-SHA256 of
// timestamp + "." + raw body, returned as Productive-Signature t=, s=.
func signWebhookBody(secret, timestamp string, body []byte) string {
	return "t=" + timestamp + ", s=" + webhookSignatureHex([]byte(secret), timestamp, body)
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

func taskWebhookSecret() string {
	return "created-token\nupdated-token"
}

func Test__OnTask__HandleWebhook(t *testing.T) {
	trigger := &OnTask{}

	t.Run("missing signature -> error", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: createdTaskConfiguration(),
			Body:          taskWebhookBody("91", "1", "Fix payment retries"),
			Webhook:       &contexts.NodeWebhookContext{Secret: taskWebhookSecret()},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "missing")
	})

	t.Run("invalid signature -> error", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", "t=1710000000, s="+strings.Repeat("ab", 32)),
			Configuration: createdTaskConfiguration(),
			Body:          taskWebhookBody("91", "1", "Fix payment retries"),
			Webhook:       &contexts.NodeWebhookContext{Secret: taskWebhookSecret()},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook signature")
	})

	t.Run("event not in configured actions -> ignored", func(t *testing.T) {
		body := taskWebhookBody("91", "1", "Fix payment retries")
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionCreated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("updated-token", "1710000000", body)),
			Configuration: configuration,
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: taskWebhookSecret()},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Zero(t, events.Count())
	})

	t.Run("valid task.created delivery without an event header -> emits the envelope", func(t *testing.T) {
		body := taskWebhookBody("20295734", "1049891", "webhook test")
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1049891", "actions": []string{ActionCreated}}

		code, response, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("created-token", "1710000000", body)),
			Configuration: configuration,
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: taskWebhookSecret()},
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
		assert.Equal(t, "20295734", document["id"])
	})

	t.Run("labeled signature tokens identify task.updated without an event header", func(t *testing.T) {
		body := taskWebhookBody("91", "1", "Fix payment retries")
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionUpdated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("updated-token", "1710000000", body)),
			Configuration: configuration,
			Body:          body,
			Webhook: &contexts.NodeWebhookContext{
				Secret: TaskCreatedEvent + "=created-token\n" + TaskUpdatedEvent + "=updated-token",
			},
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		envelope, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		meta, ok := envelope["meta"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, TaskUpdatedEvent, meta["event"])
	})

	t.Run("one legacy token does not classify an update as created", func(t *testing.T) {
		body := taskWebhookBody("91", "1", "Fix payment retries")
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("shared-token", "1710000000", body)),
			Configuration: createdTaskConfiguration(),
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "shared-token"},
			Events:        events,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "missing")
		assert.Zero(t, events.Count())
	})

	t.Run("shared signature token uses the event query", func(t *testing.T) {
		body := taskWebhookBody("91", "1", "Fix payment retries")
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionUpdated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("shared-token", "1710000000", body)),
			Query:         map[string][]string{"event": {TaskUpdatedEvent}},
			Configuration: configuration,
			Body:          body,
			Webhook: &contexts.NodeWebhookContext{
				Secret: TaskCreatedEvent + "=shared-token\n" + TaskUpdatedEvent + "=shared-token",
			},
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		envelope, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		meta, ok := envelope["meta"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, TaskUpdatedEvent, meta["event"])
	})

	t.Run("delivery for another project -> ignored", func(t *testing.T) {
		body := taskWebhookBody("91", "other-project", "Fix payment retries")
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, signWebhookBody("s3cr3t", "1710000000", body)),
			Configuration: createdTaskConfiguration(),
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Zero(t, events.Count())
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
