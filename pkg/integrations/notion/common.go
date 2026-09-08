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
	// as reported by Notion. The next poll only emits pages created after it.
	// Notion timestamps are used rather than local clock reads, because the
	// two drift and a drifting cursor either repeats or skips pages.
	PolledUntil string `json:"polledUntil,omitempty" mapstructure:"polledUntil,omitempty"`
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
