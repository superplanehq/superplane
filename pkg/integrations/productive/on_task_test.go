package productive

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

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

func taskWebhookBodyWithList(id, projectID, title, listID, created string) []byte {
	body, _ := json.Marshal(map[string]any{
		"event":     "update_task",
		"item_type": "task",
		"item_id":   id,
		"created":   created,
		"object": map[string]any{
			"data": map[string]any{
				"id":   id,
				"type": "tasks",
				"attributes": map[string]any{
					"title":   title,
					"type_id": 1,
				},
				"relationships": map[string]any{
					"project": map[string]any{
						"data": map[string]any{"type": "projects", "id": projectID},
					},
					"task_list": map[string]any{
						"data": map[string]any{"type": "task_lists", "id": listID},
					},
				},
			},
		},
	})
	return body
}

// updatedTaskDelivery is a task.updated body whose title edit matches
// updatedTaskActivity. Signature tests use it so routing still emits.
func updatedTaskDelivery(id, projectID, title string, created time.Time) []byte {
	return taskWebhookBodyWithList(id, projectID, title, "20", created.Format(time.RFC3339Nano))
}

func updatedTaskActivity(created time.Time) *http.Response {
	return activitiesResponse([]activityRecord{{
		id:        "1",
		at:        created,
		changeset: map[string]any{"title": []any{"Previous title", "Fix payment retries"}},
	}})
}

// taskWebhookBody builds the envelope Productive.io posts for a task webhook.
// The task resource is the JSON:API document under object.data.
func taskWebhookBody(id, projectID, title string) []byte {
	body, _ := json.Marshal(map[string]any{
		"event":     "create_task",
		"item_type": "task",
		"item_id":   id,
		"object": map[string]any{
			"data": map[string]any{
				"id":   id,
				"type": "tasks",
				"attributes": map[string]any{
					"title":   title,
					"type_id": 1,
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
			Integration:   integrationWithProject(),
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
		assert.Equal(t, "https://app.productive.io/org-1/tasks/20295734", envelope["url"])
	})

	t.Run("labeled signature tokens identify task.updated without an event header", func(t *testing.T) {
		created := testClock()
		body := updatedTaskDelivery("91", "1", "Fix payment retries", created)
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionUpdated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("updated-token", "1710000000", body)),
			Configuration: configuration,
			Body:          body,
			Webhook: &contexts.NodeWebhookContext{
				Secret: TaskCreatedEvent + "=created-token\n" + TaskUpdatedEvent + "=updated-token",
			},
			Events:      events,
			HTTP:        &contexts.HTTPContext{Responses: []*http.Response{updatedTaskActivity(created)}},
			Integration: integrationWithProject(),
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

	t.Run("one legacy token with an updated query emits an update", func(t *testing.T) {
		created := testClock()
		body := updatedTaskDelivery("91", "1", "Fix payment retries", created)
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionUpdated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("shared-token", "1710000000", body)),
			Query:         map[string][]string{"event": {TaskUpdatedEvent}},
			Configuration: configuration,
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "shared-token"},
			Events:        events,
			HTTP:          &contexts.HTTPContext{Responses: []*http.Response{updatedTaskActivity(created)}},
			Integration:   integrationWithProject(),
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
		created := testClock()
		body := updatedTaskDelivery("91", "1", "Fix payment retries", created)
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
			Events:      events,
			HTTP:        &contexts.HTTPContext{Responses: []*http.Response{updatedTaskActivity(created)}},
			Integration: integrationWithProject(),
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

	t.Run("production task.created payload with a shared token emits", func(t *testing.T) {
		body, err := os.ReadFile("testdata/task_created_delivery.json")
		require.NoError(t, err)
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1049891", "actions": []string{ActionCreated}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders("", signWebhookBody("shared-token", "1710000000", body)),
			Query:         map[string][]string{"event": {TaskCreatedEvent}},
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
		document, ok := envelope["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "20305431", document["id"])
		attributes, ok := document["attributes"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "webhook test 100000", attributes["title"])
	})

	t.Run("json api task without the delivery envelope -> missing task data", func(t *testing.T) {
		body := []byte(`{"data":{"id":"91","type":"tasks","attributes":{"title":"Fix payment retries"}}}`)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, signWebhookBody("s3cr3t", "1710000000", body)),
			Configuration: createdTaskConfiguration(),
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "missing task data")
	})

	t.Run("task list fetch failure returns error so the webhook retries", func(t *testing.T) {
		body := taskWebhookBody("91", "1", "Fix payment retries")
		events := &contexts.EventContext{}
		unavailable := &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Unavailable"}]}`)),
		}
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			unavailable, unavailable, unavailable,
		}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, signWebhookBody("s3cr3t", "1710000000", body)),
			Configuration: createdTaskConfiguration(),
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
			HTTP:          httpContext,
			Integration:   integrationWithProject(),
		})

		assert.Equal(t, http.StatusInternalServerError, code)
		require.ErrorContains(t, err, "task list unavailable")
		assert.Zero(t, events.Count())
		require.Len(t, httpContext.Requests, taskListFetchAttempts)
	})

	t.Run("task list fetch succeeds on retry", func(t *testing.T) {
		body := taskWebhookBody("91", "1", "Fix payment retries")
		events := &contexts.EventContext{}
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Unavailable"}]}`)),
			},
			jsonResponse(`{"data":{
				"id":"91",
				"type":"tasks",
				"attributes":{"task_number":512,"title":"Fix payment retries"},
				"relationships":{
					"project":{"data":{"type":"projects","id":"1"}},
					"task_list":{"data":{"type":"task_lists","id":"list-bugs"}}
				}
			}}`),
		}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskCreatedEvent, signWebhookBody("s3cr3t", "1710000000", body)),
			Configuration: createdTaskConfiguration(),
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
			HTTP:          httpContext,
			Integration:   integrationWithProject(),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		envelope, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		document, ok := envelope["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "list-bugs", taskListID(document))
	})

	t.Run("task update uses the activity for this delivery, not a later edit", func(t *testing.T) {
		delivered := testClock()
		later := delivered.Add(30 * time.Second)
		activities := []activityRecord{
			{id: "2", at: later, changeset: map[string]any{"title": []any{"Fix payment retries", "Renamed"}}},
			{id: "1", at: delivered, changeset: map[string]any{"task_list_id": []any{float64(10), float64(20)}}},
		}
		moveBody := taskWebhookBodyWithList("91", "1", "Fix payment retries", "20", delivered.Format(time.RFC3339Nano))
		events := &contexts.EventContext{}
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{activitiesResponse(activities)}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskUpdatedEvent, signWebhookBody("s3cr3t", "1710000000", moveBody)),
			Configuration: map[string]any{"project": "1", "actions": []string{ActionUpdated}},
			Body:          moveBody,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
			HTTP:          httpContext,
			Integration:   integrationWithProject(),
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		envelope, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		meta, ok := envelope["meta"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, map[string]any{"from": "10", "to": "20"}, meta["task_list_move"])

		editBody := taskWebhookBodyWithList("91", "1", "Renamed", "20", later.Format(time.RFC3339Nano))
		editEvents := &contexts.EventContext{}
		editHTTP := &contexts.HTTPContext{Responses: []*http.Response{activitiesResponse(activities)}}
		code, _, err = trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskUpdatedEvent, signWebhookBody("s3cr3t", "1710000000", editBody)),
			Configuration: map[string]any{"project": "1", "actions": []string{ActionUpdated}},
			Body:          editBody,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        editEvents,
			HTTP:          editHTTP,
			Integration:   integrationWithProject(),
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, editEvents.Count())
		editEnvelope, ok := editEvents.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		editMeta, ok := editEnvelope["meta"].(map[string]any)
		require.True(t, ok)
		_, hasMove := editMeta["task_list_move"]
		assert.False(t, hasMove)
	})

	t.Run("task update is retried when the activity lookup fails", func(t *testing.T) {
		body := taskWebhookBodyWithList("91", "1", "Fix payment retries", "20", testClock().Format(time.RFC3339Nano))
		events := &contexts.EventContext{}
		unavailable := &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Unavailable"}]}`)),
		}
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			unavailable, unavailable, unavailable,
		}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskUpdatedEvent, signWebhookBody("s3cr3t", "1710000000", body)),
			Configuration: map[string]any{"project": "1", "actions": []string{ActionUpdated}},
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
			HTTP:          httpContext,
			Integration:   integrationWithProject(),
		})

		assert.Equal(t, http.StatusInternalServerError, code)
		require.ErrorContains(t, err, "task update activity unavailable")
		assert.Zero(t, events.Count())
		require.Len(t, httpContext.Requests, taskListFetchAttempts)
	})

	t.Run("task update is retried when only a later activity is available", func(t *testing.T) {
		delivered := testClock()
		body := taskWebhookBodyWithList("91", "1", "Fix payment retries", "20", delivered.Format(time.RFC3339Nano))
		events := &contexts.EventContext{}
		laterActivity := func() *http.Response {
			return activitiesResponse([]activityRecord{{
				id:        "9",
				at:        delivered.Add(10 * time.Minute),
				changeset: map[string]any{"task_list_id": []any{float64(20), float64(30)}},
			}})
		}
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{laterActivity(), laterActivity(), laterActivity()}}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       webhookHeaders(TaskUpdatedEvent, signWebhookBody("s3cr3t", "1710000000", body)),
			Configuration: map[string]any{"project": "1", "actions": []string{ActionUpdated}},
			Body:          body,
			Webhook:       &contexts.NodeWebhookContext{Secret: "s3cr3t"},
			Events:        events,
			HTTP:          httpContext,
			Integration:   integrationWithProject(),
		})

		assert.Equal(t, http.StatusInternalServerError, code)
		require.ErrorContains(t, err, "task update activity unavailable")
		assert.Zero(t, events.Count())
		require.Len(t, httpContext.Requests, taskListFetchAttempts)
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
