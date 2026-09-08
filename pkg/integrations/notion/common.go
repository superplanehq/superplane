package notion

const (
	// BaseURL is Notion's REST API base URL.
	BaseURL = "https://api.notion.com/v1"

	// APIVersion pins the Notion API version this client speaks. Notion
	// requires it on every request, and can change behavior on newer
	// versions, so pinning keeps requests stable across account upgrades.
	APIVersion = "2022-06-28"

	// ResourceTypeDatabase is the resource type databases are listed and
	// picked as, both for the onPageAdded trigger's database field and
	// ListResources.
	ResourceTypeDatabase = "database"

	// PagePayloadType is the payload type the onPageAdded trigger emits.
	PagePayloadType = "notion.page"

	// PageCreatedEvent names the change a page event carries, in the "meta"
	// object of the emitted envelope. Synchronizing later page edits is out
	// of scope, so this is the only event the trigger ever reports.
	PageCreatedEvent = "page.created"
)

// NodeMetadata is stored on notion.onPageAdded nodes, so canvas cards can show
// the database without re-querying Notion, and so each poll knows where the
// one before it stopped.
type NodeMetadata struct {
	Database *Database `json:"database,omitempty" mapstructure:"database,omitempty"`

	// PolledUntil is the created time of the newest page the trigger emitted,
	// as reported by Notion. The next poll re-reads that time and everything
	// after it. Notion timestamps are used rather than local clock reads,
	// because the two drift and a drifting cursor either repeats or skips pages.
	PolledUntil string `json:"polledUntil,omitempty" mapstructure:"polledUntil,omitempty"`

	// EmittedAtCursor holds the ids of the pages already handled whose created
	// time equals PolledUntil. Notion reports created_time only to the minute,
	// so several pages can share the cursor's minute. The next poll re-reads
	// that minute (created_time on_or_after PolledUntil) and skips these ids, so
	// a page sharing an emitted page's minute - an overflow page or one whose
	// content could not be read yet - is neither skipped nor emitted twice.
	EmittedAtCursor []string `json:"emittedAtCursor,omitempty" mapstructure:"emittedAtCursor,omitempty"`

	// FailedContentPageID and FailedContentAttempts track the oldest unread
	// page whose content read keeps failing. A content read is usually
	// transient, so the page is retried on later polls rather than emitted
	// without its body; these fields count the consecutive polls that failed on
	// the same page so a page whose content never loads is eventually emitted
	// without content instead of stalling every page behind it forever.
	FailedContentPageID   string `json:"failedContentPageId,omitempty" mapstructure:"failedContentPageId,omitempty"`
	FailedContentAttempts int    `json:"failedContentAttempts,omitempty" mapstructure:"failedContentAttempts,omitempty"`
}

// PageEnvelope wraps a page the way every consumer of this trigger reads it:
// the simplified page under "data", and the change that produced it under
// "meta". Seeded pages use the same shape, so nothing downstream can tell a
// seeded page from a polled one.
func PageEnvelope(event string, page map[string]any) map[string]any {
	return map[string]any{
		"meta": map[string]any{"event": event},
		"data": page,
	}
}
