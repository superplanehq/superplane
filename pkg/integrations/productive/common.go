package productive

const (
	// BaseURL is Productive.io's API v2 base URL, used unless the "region"
	// configuration field overrides it.
	BaseURL = "https://api.productive.io/api/v2"

	// AuthTokenHeader carries the Productive.io API token on every request.
	AuthTokenHeader = "X-Auth-Token"

	// OrganizationIDHeader scopes every request to one Productive.io
	// organization. Productive.io issues tokens per person, not per
	// organization, so this header is required on every call.
	OrganizationIDHeader = "X-Organization-Id"

	// TaskPayloadType is the payload type the onTask trigger emits.
	TaskPayloadType = "productive.task"

	// ResourceTypeProject is the resource type projects are listed and picked
	// as, both for the onTask trigger's project field and ListResources.
	ResourceTypeProject = "project"

	// EventHeader carries the webhook event name Productive.io sends with
	// each delivery, e.g. "task.created".
	EventHeader = "X-Productive-Event"

	// SignatureHeader carries a hex-encoded HMAC-SHA256 of the raw request
	// body, signed with the secret given when the webhook was created.
	SignatureHeader = "X-Productive-Signature"

	// TaskCreatedEvent and TaskUpdatedEvent are the webhook event names
	// Productive.io sends for task lifecycle changes.
	TaskCreatedEvent = "task.created"
	TaskUpdatedEvent = "task.updated"
)

// NodeMetadata is stored on productive.onTask nodes at setup time, so canvas
// cards can show the project without re-querying Productive.io.
type NodeMetadata struct {
	Project *Project `json:"project,omitempty" mapstructure:"project,omitempty"`
}

// actionEvents maps the onTask "actions" configuration values to the webhook
// event names Productive.io sends, so the trigger can request only the
// events it was configured to listen for.
var actionEvents = map[string]string{
	"created": TaskCreatedEvent,
	"updated": TaskUpdatedEvent,
}

// eventsForActions translates configured actions into webhook event names.
// An action with no known event is dropped rather than rejected, so a future
// action value added to the multi-select cannot break an existing trigger.
func eventsForActions(actions []string) []string {
	events := make([]string, 0, len(actions))
	for _, action := range actions {
		if event, ok := actionEvents[action]; ok {
			events = append(events, event)
		}
	}
	return events
}
