package factories

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// Work order event type strings, mirroring pkg/models/factory/events.go.
// Kept as local constants (rather than importing the server-side package)
// since the CLI only needs the wire-format string, not the Go event types.
const (
	eventTypeOrderAssigneesUpdated = "order.assignees.updated"
	eventTypeOrderStatusUpdated    = "order.status.updated"
	eventTypeOrderCommentAdded     = "order.comment.added"
	eventTypeOrderArtifactAdded    = "order.artifact.added"
	eventTypeStepExecutionCreated  = "step.execution.created"
	eventTypeStepExecutionFinished = "step.execution.finished"
)

func formatOrderState(state openapi_client.FactoriesWorkOrderState) string {
	switch state {
	case openapi_client.FACTORIESWORKORDERSTATE_STATE_DRAFT:
		return "Draft"
	case openapi_client.FACTORIESWORKORDERSTATE_STATE_OPEN:
		return "Open"
	case openapi_client.FACTORIESWORKORDERSTATE_STATE_CLOSED:
		return "Closed"
	default:
		return "-"
	}
}

func formatOrderResult(result openapi_client.FactoriesWorkOrderResult) string {
	switch result {
	case openapi_client.FACTORIESWORKORDERRESULT_RESULT_COMPLETED:
		return "Completed"
	case openapi_client.FACTORIESWORKORDERRESULT_RESULT_REJECTED:
		return "Rejected"
	case openapi_client.FACTORIESWORKORDERRESULT_RESULT_FAILED:
		return "Failed"
	default:
		return "-"
	}
}

func formatRelativeTime(value time.Time) string {
	return formatRelativeTimeAt(value, time.Now())
}

func formatRelativeTimeAt(value time.Time, now time.Time) string {
	if value.IsZero() {
		return "-"
	}

	elapsed := now.Sub(value)
	if elapsed < 0 {
		elapsed = 0
	}

	switch {
	case elapsed < time.Minute:
		seconds := int(elapsed.Seconds())
		if seconds <= 1 {
			return "1s ago"
		}
		return fmt.Sprintf("%ds ago", seconds)
	case elapsed < time.Hour:
		minutes := int(elapsed.Minutes())
		if minutes <= 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	case elapsed < 24*time.Hour:
		hours := int(elapsed.Hours())
		if hours <= 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(elapsed.Hours() / 24)
		if days <= 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

// formatAssigneeList renders a work order's assignees as a single
// comma-joined line of names (falling back to id when a name is missing),
// for use in tabular "list" output. Returns "-" when there are none.
func formatAssigneeList(assignees []openapi_client.SuperplaneFactoriesUserRef) string {
	if len(assignees) == 0 {
		return "-"
	}

	names := make([]string, 0, len(assignees))
	for _, assignee := range assignees {
		names = append(names, userRefLabel(assignee))
	}
	return strings.Join(names, ", ")
}

// formatUserRef renders a single user reference as "name (id)", falling
// back to just the name or just the id when the other is missing, and "-"
// when both are missing.
func formatUserRef(ref openapi_client.SuperplaneFactoriesUserRef) string {
	name := ref.GetName()
	id := ref.GetId()
	switch {
	case name != "" && id != "":
		return fmt.Sprintf("%s (%s)", name, id)
	case name != "":
		return name
	case id != "":
		return id
	default:
		return "-"
	}
}

func userRefLabel(ref openapi_client.SuperplaneFactoriesUserRef) string {
	if name := ref.GetName(); name != "" {
		return name
	}
	return ref.GetId()
}

// formatWorkOrderCreator renders the polymorphic `createdBy` field: either a
// resolved user (as "name (id)") or a canvas automation (as "automation
// (node/app name)"). Returns "-" when neither branch is populated.
func formatWorkOrderCreator(creator openapi_client.FactoriesWorkOrderCreator) string {
	if automation, ok := creator.GetAutomationOk(); ok && automation != nil {
		name := automation.GetNodeName()
		if name == "" {
			name = automation.GetAppName()
		}
		if name == "" {
			return "automation"
		}
		return fmt.Sprintf("automation (%s)", name)
	}
	if user, ok := creator.GetUserOk(); ok && user != nil {
		return formatUserRef(*user)
	}
	return "-"
}

// unknownActorLabel is shown in place of a user ID that couldn't be resolved
// to an organization member, so raw UUIDs never leak into event output.
const unknownActorLabel = "unknown user"

// memberEmailLookup resolves organization member IDs (as they appear in raw
// event payloads) to a human-readable label, preferring email and falling
// back to display name. It's built once per "describe" call from the
// organization's member list and threaded through event formatting so no
// user ID ever reaches the terminal.
type memberEmailLookup struct {
	emailByID map[string]string
}

// newMemberEmailLookup builds a lookup from the organization's member list.
func newMemberEmailLookup(users []openapi_client.SuperplaneUsersUser) memberEmailLookup {
	emailByID := make(map[string]string, len(users))
	for _, user := range users {
		metadata := user.GetMetadata()
		id := metadata.GetId()
		if id == "" {
			continue
		}

		label := metadata.GetEmail()
		if label == "" {
			spec := user.GetSpec()
			label = spec.GetDisplayName()
		}
		if label != "" {
			emailByID[id] = label
		}
	}
	return memberEmailLookup{emailByID: emailByID}
}

// actorLabel returns the resolved email/display name for a member ID, or
// unknownActorLabel when the ID is blank or doesn't match a known member.
// This is the single place raw user IDs get scrubbed from event output.
func (l memberEmailLookup) actorLabel(userID string) string {
	if userID == "" {
		return unknownActorLabel
	}
	if label, ok := l.emailByID[userID]; ok && label != "" {
		return label
	}
	return unknownActorLabel
}

// The following types decode the generic event payload (a plain
// map[string]interface{} on the wire) into typed shapes mirroring the
// server-side event structs in pkg/models/factory/events.go. Only the
// fields the CLI renders are included.

type eventUserRef struct {
	ID string `json:"id"`
}

type eventAutomationRef struct {
	NodeName string `json:"nodeName,omitempty"`
	AppName  string `json:"appName,omitempty"`
	LineName string `json:"lineName,omitempty"`
}

type eventRunRef struct {
	ID string `json:"id"`
}

type eventArtifactRef struct {
	ID   string                 `json:"id"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

type orderStatusUpdatedEvent struct {
	User       *eventUserRef       `json:"user,omitempty"`
	Automation *eventAutomationRef `json:"automation,omitempty"`
	Run        *eventRunRef        `json:"run,omitempty"`
	FromState  string              `json:"fromState"`
	ToState    string              `json:"toState"`
	FromResult string              `json:"fromResult,omitempty"`
	ToResult   string              `json:"toResult,omitempty"`
}

type orderAssigneesUpdatedEvent struct {
	User       *eventUserRef  `json:"user,omitempty"`
	Assigned   []eventUserRef `json:"assigned,omitempty"`
	Unassigned []eventUserRef `json:"unassigned,omitempty"`
}

type commentAuthorEvent struct {
	Kind       string              `json:"kind"`
	UserID     string              `json:"userId,omitempty"`
	Automation *eventAutomationRef `json:"automation,omitempty"`
}

type orderCommentAddedEvent struct {
	Body   string              `json:"body"`
	Author *commentAuthorEvent `json:"author,omitempty"`
}

type orderArtifactAddedEvent struct {
	Artifact   *eventArtifactRef   `json:"artifact,omitempty"`
	User       *eventUserRef       `json:"user,omitempty"`
	Automation *eventAutomationRef `json:"automation,omitempty"`
}

type stepExecutionEvent struct {
	StepName string `json:"stepName"`
	Line     *struct {
		Name string `json:"name"`
	} `json:"line,omitempty"`
}

// decodeEventPayload round-trips a generic event payload map through JSON
// into a typed struct. Event payloads come back from the API as a plain
// map[string]interface{} (they're stored as JSONB server-side), so this is
// the simplest way to get typed access without depending on server-internal
// packages.
func decodeEventPayload[T any](payload map[string]interface{}) (*T, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// formatCommentAuthor renders a comment/event author as either the
// resolved member email (never a raw user id) or "automation (<node/app
// name>)".
func formatCommentAuthor(author *commentAuthorEvent, lookup memberEmailLookup) string {
	if author == nil {
		return "unknown"
	}

	switch author.Kind {
	case "user":
		return lookup.actorLabel(author.UserID)
	case "automation":
		return formatAutomationActor(author.Automation)
	default:
		return "unknown"
	}
}

func formatAutomationActor(automation *eventAutomationRef) string {
	name := ""
	if automation != nil {
		name = automation.NodeName
		if name == "" {
			name = automation.AppName
		}
	}
	if name == "" {
		return "automation"
	}
	return fmt.Sprintf("automation (%s)", name)
}

// resolveActor renders "who/what" caused an event, preferring the user,
// then automation, then originating run, in that order. User IDs are always
// resolved through the lookup rather than printed raw; an originating run
// (an internal implementation detail) is described generically rather than
// by its ID.
func resolveActor(user *eventUserRef, automation *eventAutomationRef, run *eventRunRef, lookup memberEmailLookup) string {
	switch {
	case user != nil && user.ID != "":
		return lookup.actorLabel(user.ID)
	case automation != nil:
		return formatAutomationActor(automation)
	case run != nil && run.ID != "":
		return "an automated process"
	default:
		return ""
	}
}

// decodeCommentEvent extracts the author label and body from a
// order.comment.added event, for use by both the Comments and Events
// sections of "describe".
func decodeCommentEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) (author string, body string, ok bool) {
	data, err := decodeEventPayload[orderCommentAddedEvent](event.GetEvent())
	if err != nil {
		return "", "", false
	}
	return formatCommentAuthor(data.Author, lookup), data.Body, true
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// describeEvent renders a single human-readable line (the MESSAGE column)
// for a work order event, covering all known event types with a generic
// fallback (type + raw JSON) for anything else, so new event types don't
// break the command. User IDs found in the payload are resolved to member
// emails via lookup rather than printed raw.
func describeEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) string {
	switch event.GetType() {
	case eventTypeOrderStatusUpdated:
		return describeStatusUpdatedEvent(event, lookup)
	case eventTypeOrderAssigneesUpdated:
		return describeAssigneesUpdatedEvent(event, lookup)
	case eventTypeOrderCommentAdded:
		return describeCommentAddedEvent(event, lookup)
	case eventTypeOrderArtifactAdded:
		return describeArtifactAddedEvent(event)
	case eventTypeStepExecutionCreated:
		return describeStepExecutionEvent(event, "started")
	case eventTypeStepExecutionFinished:
		return describeStepExecutionEvent(event, "finished")
	default:
		return describeUnknownEvent(event)
	}
}

// describeStatusTransition renders the work order status change itself
// (without the actor suffix), mirroring the web UI's
// describeStatusTransition (workOrderTimelineFromEvents.ts) so the two
// clients read consistently.
func describeStatusTransition(fromState, toState, toResult string) string {
	switch {
	case fromState == "":
		return "Work order created"
	case toState == "closed":
		if toResult != "" {
			return fmt.Sprintf("Work order closed as %s", titleCase(toResult))
		}
		return "Work order closed"
	case fromState == "draft" && toState == "open":
		return "Work order opened"
	case fromState == "open" && toState == "draft":
		return "Work order moved back to Draft"
	case fromState == "closed" && toState == "open":
		return "Work order reopened"
	default:
		return fmt.Sprintf("Work order moved from %s to %s", titleCase(fromState), titleCase(toState))
	}
}

func describeStatusUpdatedEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) string {
	data, err := decodeEventPayload[orderStatusUpdatedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	line := describeStatusTransition(data.FromState, data.ToState, data.ToResult)
	if actor := resolveActor(data.User, data.Automation, data.Run, lookup); actor != "" {
		line += " by " + actor
	}
	return line
}

func describeAssigneesUpdatedEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) string {
	data, err := decodeEventPayload[orderAssigneesUpdatedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	var parts []string
	if len(data.Assigned) > 0 {
		parts = append(parts, "Work order assigned to "+joinEventActors(data.Assigned, lookup))
	}
	if len(data.Unassigned) > 0 {
		parts = append(parts, "unassigned "+joinEventActors(data.Unassigned, lookup))
	}
	if len(parts) == 0 {
		parts = append(parts, "Work order assignees updated")
	}

	line := strings.Join(parts, "; ")
	if actor := resolveActor(data.User, nil, nil, lookup); actor != "" {
		line += " by " + actor
	}
	return line
}

func joinEventActors(users []eventUserRef, lookup memberEmailLookup) string {
	labels := make([]string, 0, len(users))
	for _, user := range users {
		labels = append(labels, lookup.actorLabel(user.ID))
	}
	return strings.Join(labels, ", ")
}

func describeCommentAddedEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) string {
	author, body, ok := decodeCommentEvent(event, lookup)
	if !ok {
		return describeUnknownEvent(event)
	}
	return fmt.Sprintf("%s commented: %s", author, body)
}

// artifactTypeName maps an artifact's raw type string to the label used in
// its event message, spelling out abbreviations ("pr" -> "PR") and passing
// anything else through unchanged.
func artifactTypeName(t string) string {
	switch t {
	case "pr":
		return "PR"
	default:
		return t
	}
}

// artifactLabel extracts the artifact's own display info (never a run or
// artifact ID) from its free-form data, using the same title/name/url
// precedence as "artifact list".
func artifactLabel(artifact *eventArtifactRef) string {
	if artifact == nil || artifact.Data == nil {
		return ""
	}
	if value, ok := artifact.Data["title"].(string); ok && value != "" {
		return value
	}
	if value, ok := artifact.Data["name"].(string); ok && value != "" {
		return value
	}
	if value, ok := artifact.Data["url"].(string); ok && value != "" {
		return value
	}
	return ""
}

// describeArtifactAddedEvent renders only the artifact's own information
// (type + label) — no run IDs, artifact IDs, or actor — per the ticket's
// request to keep artifact messages focused on the artifact itself.
//
// The fixed "<type> added" phrase is kept at the start of the message
// (rather than after the label) so it stays glued to the TYPE/AGE columns
// on screen: artifact labels (e.g. a PR title) can be long enough that a
// terminal wraps the line, and when "added" trailed the label it could get
// pushed onto its own line, reading like a stray fragment.
func describeArtifactAddedEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	data, err := decodeEventPayload[orderArtifactAddedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	if data.Artifact == nil {
		return "Artifact added"
	}

	typeName := artifactTypeName(data.Artifact.Type)
	if label := artifactLabel(data.Artifact); label != "" {
		return fmt.Sprintf("%s added: %s", typeName, label)
	}
	return fmt.Sprintf("%s added", typeName)
}

func describeStepExecutionEvent(event openapi_client.FactoriesWorkOrderEvent, verb string) string {
	data, err := decodeEventPayload[stepExecutionEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	line := fmt.Sprintf("step %q %s", data.StepName, verb)
	if data.Line != nil && data.Line.Name != "" {
		line += fmt.Sprintf(" (line: %s)", data.Line.Name)
	}
	return line
}

func describeUnknownEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	eventType := event.GetType()
	if eventType == "" {
		eventType = "unknown"
	}

	raw, err := json.Marshal(event.GetEvent())
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return eventType
	}

	return fmt.Sprintf("%s: %s", eventType, string(raw))
}
