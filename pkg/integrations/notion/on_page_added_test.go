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

// pagePageMore answers a query with one page of database pages and reports that
// more remain behind nextCursor, so tests can exercise pagination.
func pagePageMore(nextCursor string, pages ...string) *http.Response {
	return jsonResponse(fmt.Sprintf(`{"results":[%s],"has_more":true,"next_cursor":%q}`, strings.Join(pages, ","), nextCursor))
}

// pageTimestamp formats a time the way Notion reports a page's created_time,
// so tests can derive their fixtures from time.Now() rather than hardcoding
// absolute dates.
func pageTimestamp(at time.Time) string {
	return formatPageTime(at)
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
		newest := pageTimestamp(time.Now().Add(-90 * time.Minute))
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			databaseResponse(),
			pagePage(pageDocument("page-1", "Fix payment retries", newest)),
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
		assert.Equal(t, newest, stored.PolledUntil)

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
		cursor := pageTimestamp(time.Now().Add(-90 * time.Minute))
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{databaseResponse()}}
		metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: cursor}}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithDatabase(),
			HTTP:          httpContext,
			Metadata:      metadata,
			Requests:      &contexts.RequestContext{},
			Configuration: databaseConfiguration(),
		})

		require.NoError(t, err)
		assert.Equal(t, cursor, nodeMetadata(t, metadata).PolledUntil)
		assert.Len(t, httpContext.Requests, 1, "an existing cursor must not be looked up again")
	})
}

func Test__OnPageAdded__HandleHook__UnknownHook(t *testing.T) {
	_, err := (&OnPageAdded{}).HandleHook(core.TriggerHookContext{Name: "somethingElse"})
	require.ErrorContains(t, err, "not supported")
}

func Test__OnPageAdded__Poll__EmitsPagesAddedAfterTheCursor(t *testing.T) {
	now := time.Now().UTC()
	cursor := pageTimestamp(now.Add(-3 * time.Hour))
	older := pageTimestamp(now.Add(-2 * time.Hour))
	newest := pageTimestamp(now.Add(-time.Hour))
	beforeCursor := pageTimestamp(now.Add(-5 * time.Hour))

	//
	// The poll reads pages oldest first, so the fixture lists them that way.
	// The page created before the cursor is a defensive case: Notion filters it
	// out server-side, and the trigger drops it too rather than emitting it.
	//
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-91", "Already reported", beforeCursor),
			pageDocument("page-92", "Older", older),
			pageDocument("page-93", "Newest", newest),
		),
		blocksResponse("Body of Older"),
		blocksResponse("Body of Newest"),
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: cursor}}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	// Oldest first, so the newest page ends up at the top of the backlog.
	assert.Equal(t, []string{"Older", "Newest"}, emittedTitles(t, events))
	assert.Equal(t, newest, nodeMetadata(t, metadata).PolledUntil)

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
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: pageTimestamp(time.Now().Add(-time.Hour))}}

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
	cursor := pageTimestamp(time.Now().Add(-time.Hour))
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: cursor}}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, requests,
	))

	require.NoError(t, err)
	assert.Zero(t, events.Count())
	assert.Equal(t, pollPagesHook, requests.Action)
	assert.Equal(t, cursor, nodeMetadata(t, metadata).PolledUntil, "a failed read must not move the cursor")
}

func Test__OnPageAdded__Poll__StopsAtAPageWhoseContentFailsToRead(t *testing.T) {
	now := time.Now().UTC()
	cursor := pageTimestamp(now.Add(-3 * time.Hour))

	// Pages arrive oldest first, so the first page emitted is "Older". Its
	// content read fails, so nothing is emitted and the cursor stays behind it.
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-91", "Older", pageTimestamp(now.Add(-2*time.Hour))),
			pageDocument("page-92", "Newest", pageTimestamp(now.Add(-time.Hour))),
		),
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"boom"}`))},
	}}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: cursor}}
	events := &contexts.EventContext{}

	_, err := (&OnPageAdded{}).HandleHook(pollContext(
		databaseConfiguration(), metadata, httpContext, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Zero(t, events.Count())
	assert.Equal(t, cursor, nodeMetadata(t, metadata).PolledUntil, "the cursor must stay behind the unread page")
}

// Regression: Notion reports created_time only to the minute, so several pages
// can share the cursor's minute. A page left unread at that minute - here one
// whose content read fails - must be caught up by the next poll, not skipped
// forever behind a strictly-after cursor.
func Test__OnPageAdded__Poll__CatchesUpAPageSharingAnEmittedPagesMinute(t *testing.T) {
	now := time.Now().UTC()
	cursor := pageTimestamp(now.Add(-3 * time.Hour))
	older := pageTimestamp(now.Add(-2 * time.Hour))
	boundary := pageTimestamp(now.Add(-time.Hour))

	trigger := &OnPageAdded{}
	metadata := &contexts.MetadataContext{Metadata: NodeMetadata{PolledUntil: cursor}}

	// First poll: three pages arrive, two of them at the same boundary minute.
	// The second boundary page's content read fails, so it is not emitted.
	firstPoll := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-a", "A", older),
			pageDocument("page-b", "B", boundary),
			pageDocument("page-c", "C", boundary),
		),
		blocksResponse("Body A"),
		blocksResponse("Body B"),
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"boom"}`))},
	}}
	firstEvents := &contexts.EventContext{}

	_, err := trigger.HandleHook(pollContext(
		databaseConfiguration(), metadata, firstPoll, firstEvents, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Equal(t, []string{"A", "B"}, emittedTitles(t, firstEvents))
	stored := nodeMetadata(t, metadata)
	assert.Equal(t, boundary, stored.PolledUntil, "the cursor advances to the boundary minute")
	assert.Equal(t, []string{"page-b"}, stored.EmittedAtCursor, "only the emitted boundary page is recorded")

	// Second poll: the cursor's minute is re-read (on_or_after), so both
	// boundary pages come back. The already-emitted one is dropped by id, and
	// the one that failed before is emitted now instead of being lost.
	secondPoll := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-b", "B", boundary),
			pageDocument("page-c", "C", boundary),
		),
		blocksResponse("Body C"),
	}}
	secondEvents := &contexts.EventContext{}

	_, err = trigger.HandleHook(pollContext(
		databaseConfiguration(), metadata, secondPoll, secondEvents, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Equal(t, []string{"C"}, emittedTitles(t, secondEvents), "the boundary page is caught up, not skipped")
	stored = nodeMetadata(t, metadata)
	assert.Equal(t, boundary, stored.PolledUntil)
	assert.ElementsMatch(t, []string{"page-b", "page-c"}, stored.EmittedAtCursor, "both boundary pages are now recorded")
}

// Regression: the first poll must not replay pages that already existed when the
// trigger was added, including pages that share the newest page's minute.
func Test__OnPageAdded__Setup__RecordsTheNewestMinuteSoTheFirstPollDoesNotReplayIt(t *testing.T) {
	now := time.Now().UTC()
	newest := pageTimestamp(now.Add(-90 * time.Minute))
	older := pageTimestamp(now.Add(-2 * time.Hour))

	trigger := &OnPageAdded{}
	metadata := &contexts.MetadataContext{}

	// Two pre-existing pages share the newest minute; a third is older.
	setup := &contexts.HTTPContext{Responses: []*http.Response{
		databaseResponse(),
		pagePage(
			pageDocument("page-1", "Newest one", newest),
			pageDocument("page-2", "Newest two", newest),
			pageDocument("page-3", "Older", older),
		),
	}}

	err := trigger.Setup(core.TriggerContext{
		Integration:   integrationWithDatabase(),
		HTTP:          setup,
		Metadata:      metadata,
		Requests:      &contexts.RequestContext{},
		Configuration: databaseConfiguration(),
	})
	require.NoError(t, err)

	stored := nodeMetadata(t, metadata)
	assert.Equal(t, newest, stored.PolledUntil)
	assert.ElementsMatch(t, []string{"page-1", "page-2"}, stored.EmittedAtCursor,
		"every page sharing the newest minute is recorded so it is not replayed")

	// The first poll re-reads the newest minute and finds only those two pages;
	// both are already recorded, so nothing is reported as new.
	poll := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-1", "Newest one", newest),
			pageDocument("page-2", "Newest two", newest),
		),
	}}
	events := &contexts.EventContext{}

	_, err = trigger.HandleHook(pollContext(
		databaseConfiguration(), metadata, poll, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Zero(t, events.Count(), "pages that existed before the trigger must not be replayed")
}

// Regression: Notion reports created_time only to the minute, so more pages than
// a single request returns can share the newest minute. Setup must page through
// the whole newest minute and record every id, or the first poll re-reads that
// minute and emits the pages it never recorded as newly created.
func Test__OnPageAdded__Setup__RecordsEveryPageAtTheNewestMinuteAcrossPages(t *testing.T) {
	now := time.Now().UTC()
	newest := pageTimestamp(now.Add(-90 * time.Minute))
	older := pageTimestamp(now.Add(-2 * time.Hour))

	trigger := &OnPageAdded{}
	metadata := &contexts.MetadataContext{}

	// The newest minute holds more pages than a single request returns. The first
	// request fills with pages at the newest minute and reports more; the second
	// returns the rest of that minute and then an older page that ends it.
	setup := &contexts.HTTPContext{Responses: []*http.Response{
		databaseResponse(),
		pagePageMore("cursor-1",
			pageDocument("page-1", "Newest one", newest),
			pageDocument("page-2", "Newest two", newest),
		),
		pagePage(
			pageDocument("page-3", "Newest three", newest),
			pageDocument("page-4", "Older", older),
		),
	}}

	err := trigger.Setup(core.TriggerContext{
		Integration:   integrationWithDatabase(),
		HTTP:          setup,
		Metadata:      metadata,
		Requests:      &contexts.RequestContext{},
		Configuration: databaseConfiguration(),
	})
	require.NoError(t, err)

	stored := nodeMetadata(t, metadata)
	assert.Equal(t, newest, stored.PolledUntil)
	assert.ElementsMatch(t, []string{"page-1", "page-2", "page-3"}, stored.EmittedAtCursor,
		"every page sharing the newest minute is recorded across pages")

	// The first poll re-reads the newest minute and finds those three pages; all
	// are already recorded, so none is replayed as new.
	poll := &contexts.HTTPContext{Responses: []*http.Response{
		pagePage(
			pageDocument("page-1", "Newest one", newest),
			pageDocument("page-2", "Newest two", newest),
			pageDocument("page-3", "Newest three", newest),
		),
	}}
	events := &contexts.EventContext{}

	_, err = trigger.HandleHook(pollContext(
		databaseConfiguration(), metadata, poll, events, &contexts.RequestContext{},
	))
	require.NoError(t, err)

	assert.Zero(t, events.Count(), "pages that existed before the trigger must not be replayed")
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
