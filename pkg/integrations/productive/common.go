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

	// TaskCreatedEvent and TaskUpdatedEvent name the change a task event
	// carries, in the "meta" object of the emitted envelope.
	TaskCreatedEvent = "task.created"
	TaskUpdatedEvent = "task.updated"

	// ActionCreated and ActionUpdated are the values of the onTask trigger's
	// "actions" field.
	ActionCreated = "created"
	ActionUpdated = "updated"
)

// NodeMetadata is stored on productive.onTask nodes, so canvas cards can show
// the project without re-querying Productive.io, and so each poll knows where
// the one before it stopped.
type NodeMetadata struct {
	Project *Project `json:"project,omitempty" mapstructure:"project,omitempty"`

	// PolledUntil is the newest task change the trigger emitted, as reported
	// by Productive.io. The next poll only emits tasks changed after it.
	// Productive.io timestamps are used rather than local clock reads, because
	// the two drift and a drifting cursor either repeats or skips tasks.
	PolledUntil string `json:"polledUntil,omitempty" mapstructure:"polledUntil,omitempty"`
}

// TaskEnvelope wraps a task resource the way every consumer of this trigger
// reads it: the JSON:API resource under "data", and the change that produced
// it under "meta". Seeded tasks use the same shape, so nothing downstream can
// tell a seeded task from a polled one.
func TaskEnvelope(event string, document map[string]any) map[string]any {
	return map[string]any{
		"meta": map[string]any{"event": event},
		"data": document,
	}
}
