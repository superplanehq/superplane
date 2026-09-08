package notion

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

func integrationWithDatabase() *contexts.IntegrationContext {
	return testIntegration(nil)
}

func databaseResponse() *http.Response {
	return jsonResponse(`{"id":"db-1","object":"database","title":[{"plain_text":"Tasks"}]}`)
}

// pagePage answers a query with one page of database pages.
func pagePage(pages ...string) *http.Response {
	return jsonResponse(fmt.Sprintf(`{"results":[%s],"has_more":false}`, strings.Join(pages, ",")))
}

func pageDocument(id, title, createdAt string) string {
	return fmt.Sprintf(`{
		"id":%q,
		"created_time":%q,
		"url":"https://www.notion.so/%s",
		"properties":{"Name":{"type":"title","title":[{"plain_text":%q}]}}
	}`, id, createdAt, id, title)
}

// blocksResponse answers a page's content read with a single paragraph.
func blocksResponse(text string) *http.Response {
	return jsonResponse(fmt.Sprintf(`{"results":[{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":%q}]}}]}`, text))
}

func pollContext(
	configuration map[string]any,
	metadata *contexts.MetadataContext,
	httpContext *contexts.HTTPContext,
	events *contexts.EventContext,
	requests *contexts.RequestContext,
) core.TriggerHookContext {
	return core.TriggerHookContext{
		Name:          pollPagesHook,
		Configuration: configuration,
		Integration:   integrationWithDatabase(),
		HTTP:          httpContext,
		Metadata:      metadata,
		Events:        events,
		Requests:      requests,
		Logger:        log.NewEntry(log.New()),
	}
}

func databaseConfiguration() map[string]any {
	return map[string]any{"database": "db-1"}
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
		assert.Equal(t, PagePayloadType, payload.Type)

		envelope, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		page, ok := envelope["data"].(map[string]any)
		require.True(t, ok)

		title, ok := page["title"].(string)
		require.True(t, ok)
		titles = append(titles, title)
	}

	return titles
}

func Test__OnPageAdded__Setup(t *testing.T) {
	trigger := &OnPageAdded{}

	t.Run("missing database -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithDatabase(),
			HTTP:          &contexts.HTTPContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "database is required")
	})

	t.Run("unknown database -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"message":"Not found"}`))},
		}}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithDatabase(),
			HTTP:          httpContext,
			Metadata:      &contexts.MetadataContext{},
			Configuration: databaseConfiguration(),
		})

		require.ErrorContains(t, err, "error finding database")
	})

	// Pages that exist before the trigger does are not news. The first setup
	// starts at the newest page the database already carries.
	t.Run("starts polling at the database's newest page", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			databaseResponse(),
			pagePage(pageDocument("page-1", "Fix payment retries", "2026-01-03T11:30:00.000Z")),
		}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithDatabase(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      requests,
			Configuration: databaseConfiguration(),
		})

		require.NoError(t, err)

		stored := nodeMetadata(t, metadata)
		require.NotNil(t, stored.Database)
		assert.Equal(t, "Tasks", stored.Database.Name)
		assert.Equal(t, "2026-01-03T11:30:00Z", stored.PolledUntil)

		assert.Equal(t, pollPagesHook, requests.Action)
		assert.Equal(t, pollInterval, requests.Duration)
	})

	t.Run("a database without pages starts polling from now", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			databaseResponse(),
			pagePage(),
		}}
		metadata := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithDatabase(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      &contexts.RequestContext{},
			Configuration: databaseConfiguration(),
		})

		require.NoError(t, err)

		polledUntil, ok := parsePageTime(nodeMetadata(t, metadata).PolledUntil)
		require.True(t, ok)
		assert.WithinDuration(t, time.Now(), polledUntil, time.Minute)
	})

	// Setup runs again on every canvas update. Resetting the cursor there
	// would replay pages the trigger already reported.
	t.Run("keeps the cursor of an existing trigger", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{databaseResponse()}}
		metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-03T11:30:00Z"}}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithDatabase(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      &contexts.RequestContext{},
			Configuration: databaseConfiguration(),
		})

		require.NoError(t, err)
		assert.Equal(t, "2026-01-03T11:30:00Z", nodeMetadata(t, metadata).PolledUntil)
		assert.Len(t, httpContext.Requests, 1, "an existing cursor must not be looked up again")
	})
}

func Test__OnPageAdded__HandleHook__UnknownHook(t *testing.T) {
	_, err := (&OnPageAdded{}).HandleHook(core.TriggerHookContext{Name: "somethingElse"})
	require.ErrorContains(t, err, "not supported")
}

func Test__OnPageAdded__Poll__EmitsPagesAddedAfterTheCursor(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-93", "Newest", "2026-01-04T09:00:00.000Z"),
			pageDocument("page-92", "Older", "2026-01-03T09:00:00.000Z"),
			pageDocument("page-91", "Already reported", "2026-01-01T09:00:00.000Z"),
		),
		blocksResponse("Body of Older"),
		blocksResponse("Body of Newest"),
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	// Oldest first, so the newest page ends up at the top of the backlog.
	assert.Equal(t, []string{"Older", "Newest"}, emittedTitles(t, events))
	assert.Equal(t, "2026-01-04T09:00:00Z", nodeMetadata(t, metadata).PolledUntil)

	envelope, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	meta, ok := envelope["meta"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, PageCreatedEvent, meta["event"])
	page, ok := envelope["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Body of Older", page["content"])
}

func Test__OnPageAdded__Poll__SchedulesTheNextPoll(t *testing.T) {
	requests := &contexts.RequestContext{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{pagePage()}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, &contexts.EventContext{}, requests,
	))
	require.NoError(t, err)

	assert.Equal(t, pollPagesHook, requests.Action)
	assert.Equal(t, pollInterval, requests.Duration)
}

// Regression: a poll that fails must not fail the request. A failed request
// is retried at once and would poll Notion in a loop, and a poll that leaves
// no successor behind stops the trigger for good.
func Test__OnPageAdded__Poll__KeepsPollingAfterAFailedRead(t *testing.T) {
	requests := &contexts.RequestContext{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader(`{"message":"Rate limited"}`))},
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, requests,
	))

	require.NoError(t, err)
	assert.Zero(t, events.Count())
	assert.Equal(t, pollPagesHook, requests.Action)
	assert.Equal(t, "2026-01-02T09:00:00Z", nodeMetadata(t, metadata).PolledUntil, "a failed read must not move the cursor")
}

func Test__OnPageAdded__Poll__StopsAtAPageWhoseContentFailsToRead(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-92", "Newest", "2026-01-04T09:00:00.000Z"),
			pageDocument("page-91", "Older", "2026-01-03T09:00:00.000Z"),
		),
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"boom"}`))},
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: "2026-01-02T09:00:00Z"}}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Zero(t, events.Count())
	assert.Equal(t, "2026-01-02T09:00:00Z", nodeMetadata(t, metadata).PolledUntil, "the cursor must stay behind the unread page")
}

func Test__OnPageAdded__Poll__WithoutACursorReportsNothing(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	metadata := &contexts.MetadataContext{}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Zero(t, events.Count(), "a trigger with no cursor must not report the whole database as new")
	assert.Empty(t, httpContext.Requests)

	polledUntil, ok := parsePageTime(nodeMetadata(t, metadata).PolledUntil)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now(), polledUntil, time.Minute)
}

func Test__OnPageAdded__HandleWebhook__DeliversNothing(t *testing.T) {
	events := &contexts.EventContext{}

	code, body, err := (&OnPageAdded{}).HandleWebhook(core.WebhookRequestContext{
		Headers: http.Header{},
		Body:    []byte(`{"id":"page-1"}`),
		Events:  events,
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Nil(t, body)
	assert.Zero(t, events.Count())
}

func Test__OnPageAdded__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnPageAdded{}
	example := trigger.ExampleData()

	assert.Equal(t, PagePayloadType, example["type"])
	require.NotEmpty(t, example["timestamp"])

	envelope, ok := example["data"].(map[string]any)
	require.True(t, ok)

	meta, ok := envelope["meta"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, PageCreatedEvent, meta["event"])

	page, ok := envelope["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, page["title"])
	assert.NotEmpty(t, page["content"])
}
