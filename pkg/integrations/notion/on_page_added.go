package notion

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// pollPagesHook reads the pages that were added since the poll before it.
	// Notion has no webhook API for internal integrations, so this trigger
	// polls rather than subscribing.
	pollPagesHook = "pollPages"

	// pollInterval is the delay between two polls of the same database.
	pollInterval = time.Minute

	// pollPageSize is how many pages one request reads, and maxPollPages is
	// how many such requests one poll makes. Together they cap a poll well
	// above the activity a minute can hold.
	pollPageSize = 50
	maxPollPages = 5
)

type OnPageAdded struct{}

type OnPageAddedConfiguration struct {
	Database string `json:"database" mapstructure:"database"`
}

func (t *OnPageAdded) Name() string {
	return "notion.onPageAdded"
}

func (t *OnPageAdded) Label() string {
	return "On Page Added"
}

func (t *OnPageAdded) Description() string {
	return "Listen for new pages added to a Notion database"
}

func (t *OnPageAdded) Documentation() string {
	return `The On Page Added trigger starts a workflow execution when a page is added to a Notion database.

## Use Cases

- **Backlog intake**: Create a work order when a new task page is added to a database
- **Notifications**: Alert a channel when a page is added

## Configuration

- **Database** (required): Notion database to monitor. The database must be shared with the integration.

## Outputs

- **Default channel**: Emits an envelope with a ` + "`data`" + ` object that holds the page's Notion fields
  (` + "`id`" + `, ` + "`url`" + `, ` + "`properties`" + `), plus a simplified ` + "`title`" + ` and ` + "`content`" + ` (the page body as plain
  text), and a ` + "`meta.event`" + ` field that names the change (always ` + "`page.created`" + `).

## How pages arrive

The trigger reads the database every minute and emits the pages created since the read before it.
Pages that already exist when you add the trigger are not reported; only later pages are.

This trigger does not report page edits, status changes, or deletions. It only reports new pages.

Notion has no webhook API for internal integrations, so this trigger polls, at the cost of up to one
minute of delay.`
}

func (t *OnPageAdded) Icon() string {
	return "notion"
}

func (t *OnPageAdded) Color() string {
	return "gray"
}

func (t *OnPageAdded) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "database",
			Label:       "Database",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Notion database to monitor",
			Placeholder: "Select a database",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDatabase,
				},
			},
		},
	}
}

func (t *OnPageAdded) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnPageAddedConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	database, err := client.GetDatabase(config.Database)
	if err != nil {
		return fmt.Errorf("error finding database: %v", err)
	}

	metadata := NodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %v", err)
	}

	metadata.Database = database

	//
	// A trigger that started from an empty cursor would report every page it
	// can read as if it had just been added. The first setup therefore starts
	// at the newest page the database already carries, and only what happens
	// after that reaches the canvas.
	//
	if metadata.PolledUntil == "" {
		polledUntil, emittedIDs, err := newestPageBoundary(client, config.Database)
		if err != nil {
			return err
		}

		metadata.PolledUntil = polledUntil
		metadata.EmittedAtCursor = emittedIDs
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall(pollPagesHook, map[string]any{}, pollInterval)
}

func (t *OnPageAdded) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: pollPagesHook,
			Type: core.HookTypeInternal,
		},
	}
}

func (t *OnPageAdded) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	if ctx.Name != pollPagesHook {
		return nil, fmt.Errorf("hook %s not supported", ctx.Name)
	}

	return nil, t.pollPages(ctx)
}

// HandleWebhook answers the calls every trigger has to accept. This trigger
// polls instead of subscribing, so Notion delivers nothing here.
func (t *OnPageAdded) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnPageAdded) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// pollPages emits the pages of the database added since the last poll, then
// leaves the next poll behind.
func (t *OnPageAdded) pollPages(ctx core.TriggerHookContext) error {
	//
	// The next poll is scheduled before anything can fail. A poll that ends
	// early must still leave a successor behind, or one bad response would
	// stop the trigger for good.
	//
	if err := ctx.Requests.ScheduleActionCall(pollPagesHook, map[string]any{}, pollInterval); err != nil {
		return err
	}

	config, err := decodeOnPageAddedConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := NodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %v", err)
	}

	polledUntil, ok := parsePageTime(metadata.PolledUntil)
	if !ok {
		//
		// Without a usable cursor the poll cannot tell new pages from old
		// ones. Start at the current time rather than reporting the whole
		// database as new.
		//
		metadata.PolledUntil = formatPageTime(time.Now())
		return ctx.Metadata.Set(metadata)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	documents, err := changedPages(client, config.Database, polledUntil, metadata.EmittedAtCursor)
	if err != nil {
		//
		// A failed read must not fail the request: the request is retried at
		// once, which would hammer Notion while it is unhappy. The scheduled
		// poll picks the same pages up one interval later.
		//
		ctx.Logger.Errorf("Error reading the pages added to database %s: %v", config.Database, err)
		return nil
	}

	return emitChangedPages(ctx, client, documents, metadata, polledUntil)
}

// emitChangedPages emits one event per page added since the last poll, and
// moves the cursor to the newest page it handled.
func emitChangedPages(
	ctx core.TriggerHookContext,
	client *Client,
	documents []map[string]any,
	metadata NodeMetadata,
	polledUntil time.Time,
) error {
	//
	// changedPages returns the pages oldest first, so emitting them in order
	// keeps the newest at the top of a backlog and advances the cursor as it
	// goes: a page that fails leaves the cursor behind it, and a burst larger
	// than one poll can read is caught up over several polls.
	//
	// The cursor is a (time, ids) pair: PolledUntil and the ids of the pages
	// already emitted at that exact minute. Notion reports created_time only to
	// the minute, so the ids are what tell an unread page sharing an emitted
	// page's minute from a page that was already handled.
	//
	cursor := polledUntil
	emittedAtCursor := append([]string{}, metadata.EmittedAtCursor...)
	for _, document := range documents {
		createdAt, ok := pageTime(document, "created_time")
		if !ok {
			continue
		}

		page, err := BuildPageEvent(client, document)
		if err != nil {
			//
			// Stop at the first page whose content could not be read and
			// keep the cursor behind it, so the next poll re-reads from there
			// instead of skipping it. Its minute is left in the cursor, and its
			// id is not recorded, so the re-read finds it again.
			//
			ctx.Logger.Errorf("Error reading the content of Notion page %v: %v", document["id"], err)
			break
		}

		if err := ctx.Events.Emit(PagePayloadType, PageEnvelope(PageCreatedEvent, page)); err != nil {
			ctx.Logger.Errorf("Error emitting Notion page %v: %v", document["id"], err)
			break
		}

		id, _ := document["id"].(string)
		switch {
		case createdAt.After(cursor):
			//
			// A newer minute: the pages at the old minute are now strictly
			// behind the cursor, so only this page's id has to be remembered.
			//
			cursor = createdAt
			emittedAtCursor = []string{id}
		case createdAt.Equal(cursor):
			emittedAtCursor = append(emittedAtCursor, id)
		}
	}

	//
	// Persist when the cursor moved to a newer minute, or when it stayed on the
	// same minute but gained an id - the second case is how an overflow or a
	// retried page at the boundary minute is recorded without losing the pages
	// already handled there.
	//
	if cursor.After(polledUntil) || len(emittedAtCursor) != len(metadata.EmittedAtCursor) {
		metadata.PolledUntil = formatPageTime(cursor)
		metadata.EmittedAtCursor = emittedAtCursor
		return ctx.Metadata.Set(metadata)
	}

	return nil
}

// BuildPageEvent maps a page the way an intake reads it: every field Notion
// returned, plus a simplified title and content, so a template does not have
// to parse Notion's property shapes. Seeding an intake uses this too, so a
// seeded page and a polled page carry the same fields.
func BuildPageEvent(client *Client, document map[string]any) (map[string]any, error) {
	id, _ := document["id"].(string)
	content, err := client.PageContent(id)
	if err != nil {
		return nil, err
	}

	page := make(map[string]any, len(document)+2)
	for key, value := range document {
		page[key] = value
	}
	page["title"] = PageTitle(document)
	page["content"] = content

	return page, nil
}

// changedPages reads the pages of the database created at or after polledUntil,
// oldest first, and drops the ones already handled. Notion filters and sorts
// server-side, so a poll reads the oldest unreported pages first and can advance
// its cursor page by page. The cursor's own minute is re-read (on_or_after) and
// the pages already emitted at that minute are dropped by id, so a page sharing
// an emitted page's minute is caught up rather than skipped. When more pages
// were added than one poll can read, the overflow stays no older than the
// advanced cursor and is caught up by the next poll instead of skipped.
func changedPages(client *Client, databaseID string, polledUntil time.Time, emittedAtCursor []string) ([]map[string]any, error) {
	handled := make(map[string]bool, len(emittedAtCursor))
	for _, id := range emittedAtCursor {
		handled[id] = true
	}

	changed := []map[string]any{}
	cursor := ""
	createdOnOrAfter := formatPageTime(polledUntil)

	for page := 1; page <= maxPollPages; page++ {
		documents, hasMore, nextCursor, err := client.ListChangedPageDocuments(databaseID, cursor, createdOnOrAfter, pollPageSize)
		if err != nil {
			return nil, err
		}

		for _, document := range documents {
			//
			// Notion has already filtered to the cursor's minute and later. A
			// page whose timestamp cannot be read is skipped rather than
			// stopping the walk, so one odd page does not hide the pages after
			// it. Pages strictly older than the cursor are dropped defensively,
			// and pages already emitted at the cursor's minute are dropped by id
			// so the re-read of that minute does not report them again.
			//
			createdAt, ok := pageTime(document, "created_time")
			if !ok || createdAt.Before(polledUntil) {
				continue
			}

			if id, _ := document["id"].(string); handled[id] {
				continue
			}

			changed = append(changed, document)
		}

		if !hasMore || nextCursor == "" {
			return changed, nil
		}
		cursor = nextCursor
	}

	return changed, nil
}

// newestPageBoundary reports the creation time of the newest page the database
// already carries, together with the ids of every page sharing that exact
// minute, or the current time and no ids when it has no pages to read. Setup
// records these so the first poll re-reads the boundary minute (on_or_after)
// without re-emitting pages that already existed when the trigger was added.
//
// Notion reports created_time only to the minute, so more pages than a single
// request returns can share the newest minute. The boundary therefore keeps
// reading, newest first, until it reaches an older minute or the database ends,
// so every id at the newest minute is recorded and none is replayed as new.
func newestPageBoundary(client *Client, databaseID string) (string, []string, error) {
	var newest time.Time
	haveNewest := false
	ids := []string{}
	cursor := ""

	for {
		documents, hasMore, nextCursor, err := client.ListNewestPageDocuments(databaseID, cursor, pollPageSize)
		if err != nil {
			return "", nil, fmt.Errorf("error reading the pages of database %s: %v", databaseID, err)
		}

		//
		// The pages arrive newest first, so the very first page fixes the newest
		// minute and every page sharing it is at the front of the walk. Recording
		// their ids keeps the first poll from reporting a page that already
		// existed as if it had just been added.
		//
		for _, document := range documents {
			createdAt, ok := pageTime(document, "created_time")
			if !ok {
				//
				// The newest page's timestamp is what anchors the boundary. Without
				// it there is nothing to record, so start the trigger at the current
				// time as an empty database would.
				//
				if !haveNewest {
					return formatPageTime(time.Now()), nil, nil
				}
				continue
			}

			if !haveNewest {
				newest = createdAt
				haveNewest = true
			}

			if !createdAt.Equal(newest) {
				//
				// An older minute ends the boundary: every page sharing the newest
				// minute has already been seen.
				//
				return formatPageTime(newest), ids, nil
			}

			if id, _ := document["id"].(string); id != "" {
				ids = append(ids, id)
			}
		}

		if !hasMore || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	if !haveNewest {
		return formatPageTime(time.Now()), nil, nil
	}

	return formatPageTime(newest), ids, nil
}

// pageTime reads one of a page's timestamp attributes.
func pageTime(document map[string]any, attribute string) (time.Time, bool) {
	value, ok := document[attribute].(string)
	if !ok {
		return time.Time{}, false
	}

	return parsePageTime(value)
}

func parsePageTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

func formatPageTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func decodeOnPageAddedConfiguration(raw any) (OnPageAddedConfiguration, error) {
	config := OnPageAddedConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Database == "" {
		return config, fmt.Errorf("database is required")
	}

	return config, nil
}
