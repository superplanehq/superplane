package productive

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
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

// taskPage answers a changed-task read with one page of tasks.
func taskPage(tasks ...string) *http.Response {
	return jsonResponse(fmt.Sprintf(`{"data":[%s]}`, strings.Join(tasks, ",")))
}

func taskDocument(id, title, createdAt, updatedAt string) string {
	return fmt.Sprintf(`{
		"id":%q,
		"type":"tasks",
		"attributes":{"title":%q,"created_at":%q,"updated_at":%q},
		"relationships":{"project":{"data":{"type":"projects","id":"1"}}}
	}`, id, title, createdAt, updatedAt)
}

func pollContext(
	configuration map[string]any,
	metadata *contexts.MetadataContext,
	httpContext *contexts.HTTPContext,
	events *contexts.EventContext,
	requests *contexts.RequestContext,
) core.TriggerHookContext {
	return core.TriggerHookContext{
		Name:          pollTasksHook,
		Configuration: configuration,
		Integration:   integrationWithProject(),
		HTTP:          httpContext,
		Metadata:      metadata,
		Events:        events,
		Requests:      requests,
		Logger:        log.NewEntry(log.New()),
	}
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

func emittedTitles(t *testing.T, events *contexts.EventContext) []string {
	t.Helper()

	titles := []string{}
	for _, payload := range events.Payloads {
		assert.Equal(t, TaskPayloadType, payload.Type)

		envelope, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		task, ok := envelope["data"].(map[string]any)
		require.True(t, ok)
		attributes, ok := task["attributes"].(map[string]any)
		require.True(t, ok)

		title, ok := attributes["title"].(string)
		require.True(t, ok)
		titles = append(titles, title)
	}

	return titles
}

func emittedEvents(t *testing.T, events *contexts.EventContext) []string {
	t.Helper()

	names := []string{}
	for _, payload := range events.Payloads {
		envelope, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		meta, ok := envelope["meta"].(map[string]any)
		require.True(t, ok)

		name, ok := meta["event"].(string)
		require.True(t, ok)
		names = append(names, name)
	}

	return names
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

	// Tasks that exist before the trigger does are not news. The first setup
	// starts at the newest change the project already carries.
	t.Run("starts polling at the project's newest change", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			projectResponse(),
			taskPage(taskDocument("91", "Fix payment retries", "2026-01-02T10:00:00Z", "2026-01-03T11:30:00Z")),
		}}
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

		stored := nodeMetadata(t, metadata)
		require.NotNil(t, stored.Project)
		assert.Equal(t, "Payments", stored.Project.Name)
		assert.Equal(t, "2026-01-03T11:30:00Z", stored.PolledUntil)

		assert.Equal(t, pollTasksHook, requests.Action)
		assert.Equal(t, pollInterval, requests.Duration)
	})

	t.Run("a project without tasks starts polling from now", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			projectResponse(),
			taskPage(),
		}}
		metadata := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      &contexts.RequestContext{},
			Configuration: createdTaskConfiguration(),
		})

		require.NoError(t, err)

		polledUntil, ok := parseTaskTime(nodeMetadata(t, metadata).PolledUntil)
		require.True(t, ok)
		assert.WithinDuration(t, time.Now(), polledUntil, time.Minute)
	})

	// Setup runs again on every canvas update. Resetting the cursor there would
	// replay tasks the trigger already reported.
	t.Run("keeps the cursor of an existing trigger", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{projectResponse()}}
		metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-03T11:30:00Z"}}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithProject(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      &contexts.RequestContext{},
			Configuration: createdTaskConfiguration(),
		})

		require.NoError(t, err)
		assert.Equal(t, "2026-01-03T11:30:00Z", nodeMetadata(t, metadata).PolledUntil)
		assert.Len(t, httpContext.Requests, 1, "an existing cursor must not be looked up again")
	})
}

func Test__OnTask__HandleHook__UnknownHook(t *testing.T) {
	_, err := (&OnTask{}).HandleHook(core.TriggerHookContext{Name: "somethingElse"})
	require.ErrorContains(t, err, "not supported")
}

func Test__OnTask__Poll__EmitsTasksChangedAfterTheCursor(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		taskPage(
			taskDocument("93", "Newest", "2026-01-04T09:00:00Z", "2026-01-04T09:00:00Z"),
			taskDocument("92", "Older", "2026-01-03T09:00:00Z", "2026-01-03T09:00:00Z"),
			taskDocument("91", "Already reported", "2026-01-01T09:00:00Z", "2026-01-02T09:00:00Z"),
		),
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
	events := &contexts.EventContext{}

	_, err := (&OnTask{}).HandleHook(pollContext(
		createdTaskConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	// Oldest first, so the newest task ends up at the top of the backlog.
	assert.Equal(t, []string{"Older", "Newest"}, emittedTitles(t, events))
	assert.Equal(t, "2026-01-04T09:00:00Z", nodeMetadata(t, metadata).PolledUntil)

	query := httpContext.Requests[0].URL.Query()
	assert.Equal(t, "1", query.Get("filter[project_id]"))
	assert.Equal(t, "-updated_at", query.Get("sort"))
}

func Test__OnTask__Poll__NamesTheChange(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		taskPage(
			taskDocument("92", "Created after the last poll", "2026-01-04T09:00:00Z", "2026-01-04T09:00:00Z"),
			taskDocument("91", "Created earlier, changed since", "2026-01-01T09:00:00Z", "2026-01-03T09:00:00Z"),
		),
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
	events := &contexts.EventContext{}

	configuration := map[string]any{"project": "1", "actions": []string{ActionCreated, ActionUpdated}}
	_, err := (&OnTask{}).HandleHook(pollContext(
		configuration, metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Equal(t, []string{TaskUpdatedEvent, TaskCreatedEvent}, emittedEvents(t, events))
}

func Test__OnTask__Poll__FiltersActions(t *testing.T) {
	changedTaskPage := func() *contexts.HTTPContext {
		return &contexts.HTTPContext{Responses: []*http.Response{
			taskPage(taskDocument("91", "Changed since the last poll", "2026-01-01T09:00:00Z", "2026-01-03T09:00:00Z")),
		}}
	}

	t.Run("an update is skipped when only created is selected", func(t *testing.T) {
		metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
		events := &contexts.EventContext{}

		_, err := (&OnTask{}).HandleHook(pollContext(
			createdTaskConfiguration(), metadata, changedTaskPage(), events, &contexts.RequestContext{},
		))
		require.NoError(t, err)

		assert.Zero(t, events.Count())

		// The cursor still moves: the task was read, and reading it again would
		// not change the outcome.
		assert.Equal(t, "2026-01-03T09:00:00Z", nodeMetadata(t, metadata).PolledUntil)
	})

	t.Run("an update is emitted when updated is selected", func(t *testing.T) {
		metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
		events := &contexts.EventContext{}
		configuration := map[string]any{"project": "1", "actions": []string{ActionUpdated}}

		_, err := (&OnTask{}).HandleHook(pollContext(
			configuration, metadata, changedTaskPage(), events, &contexts.RequestContext{},
		))
		require.NoError(t, err)

		assert.Equal(t, 1, events.Count())
	})
}

func Test__OnTask__Poll__SchedulesTheNextPoll(t *testing.T) {
	requests := &contexts.RequestContext{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{taskPage()}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}

	_, err := (&OnTask{}).HandleHook(pollContext(
		createdTaskConfiguration(), metadata, httpContext, &contexts.EventContext{}, requests,
	))
	require.NoError(t, err)

	assert.Equal(t, pollTasksHook, requests.Action)
	assert.Equal(t, pollInterval, requests.Duration)
}

// Regression: a poll that fails must not fail the request. A failed request is
// retried at once and would poll Productive.io in a loop, and a poll that
// leaves no successor behind stops the trigger for good.
func Test__OnTask__Poll__KeepsPollingAfterAFailedRead(t *testing.T) {
	requests := &contexts.RequestContext{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader(`{"errors":[{"title":"Rate limited"}]}`))},
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
	events := &contexts.EventContext{}

	_, err := (&OnTask{}).HandleHook(pollContext(
		createdTaskConfiguration(), metadata, httpContext, events, requests,
	))

	require.NoError(t, err)
	assert.Zero(t, events.Count())
	assert.Equal(t, pollTasksHook, requests.Action)
	assert.Equal(t, "2026-01-02T09:00:00Z", nodeMetadata(t, metadata).PolledUntil, "a failed read must not move the cursor")
}

func Test__OnTask__Poll__WithoutACursorReportsNothing(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	metadata := &contexts.MetadataContext{}
	events := &contexts.EventContext{}

	_, err := (&OnTask{}).HandleHook(pollContext(
		createdTaskConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Zero(t, events.Count(), "a trigger with no cursor must not report the whole project as new")
	assert.Empty(t, httpContext.Requests)

	polledUntil, ok := parseTaskTime(nodeMetadata(t, metadata).PolledUntil)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now(), polledUntil, time.Minute)
}

func Test__OnTask__HandleWebhook__DeliversNothing(t *testing.T) {
	events := &contexts.EventContext{}

	code, body, err := (&OnTask{}).HandleWebhook(core.WebhookRequestContext{
		Headers: http.Header{},
		Body:    []byte(`{"data":{"id":"91"}}`),
		Events:  events,
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Nil(t, body)
	assert.Zero(t, events.Count())
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
