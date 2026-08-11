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
	ID   string `json:"id"`
	Type string `json:"type"`
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

// formatCommentAuthor renders a comment/event author as either the user id
// or "automation (<node/app name>)".
func formatCommentAuthor(author *commentAuthorEvent) string {
	if author == nil {
		return "unknown"
	}

	switch author.Kind {
	case "user":
		if author.UserID != "" {
			return author.UserID
		}
		return "unknown user"
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

// describeEventActor renders "who/what" caused an event, preferring the
// user, then automation, then originating run, in that order.
func describeEventActor(user *eventUserRef, automation *eventAutomationRef, run *eventRunRef) string {
	switch {
	case user != nil && user.ID != "":
		return fmt.Sprintf("user %s", user.ID)
	case automation != nil:
		return formatAutomationActor(automation)
	case run != nil && run.ID != "":
		return fmt.Sprintf("run %s", run.ID)
	default:
		return ""
	}
}

// decodeCommentEvent extracts the author label and body from a
// order.comment.added event, for use by both the Comments and Timeline
// sections of "describe".
func decodeCommentEvent(event openapi_client.FactoriesWorkOrderEvent) (author string, body string, ok bool) {
	data, err := decodeEventPayload[orderCommentAddedEvent](event.GetEvent())
	if err != nil {
		return "", "", false
	}
	return formatCommentAuthor(data.Author), data.Body, true
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// describeEvent renders a single human-readable line for a work order event,
// covering all known event types with a generic fallback (type + raw JSON)
// for anything else, so new event types don't break the command.
func describeEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	switch event.GetType() {
	case eventTypeOrderStatusUpdated:
		return describeStatusUpdatedEvent(event)
	case eventTypeOrderAssigneesUpdated:
		return describeAssigneesUpdatedEvent(event)
	case eventTypeOrderCommentAdded:
		return describeCommentAddedEvent(event)
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

func describeStatusUpdatedEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	data, err := decodeEventPayload[orderStatusUpdatedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	line := fmt.Sprintf("status changed: %s -> %s", titleCase(data.FromState), titleCase(data.ToState))
	if data.ToResult != "" {
		line += fmt.Sprintf(" (result: %s)", titleCase(data.ToResult))
	}
	if actor := describeEventActor(data.User, data.Automation, data.Run); actor != "" {
		line += " by " + actor
	}
	return line
}

func describeAssigneesUpdatedEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	data, err := decodeEventPayload[orderAssigneesUpdatedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	var parts []string
	if len(data.Assigned) > 0 {
		parts = append(parts, "assigned "+joinEventUserIDs(data.Assigned))
	}
	if len(data.Unassigned) > 0 {
		parts = append(parts, "unassigned "+joinEventUserIDs(data.Unassigned))
	}
	if len(parts) == 0 {
		parts = append(parts, "assignees updated")
	}

	line := strings.Join(parts, "; ")
	if actor := describeEventActor(data.User, nil, nil); actor != "" {
		line += " by " + actor
	}
	return line
}

func joinEventUserIDs(users []eventUserRef) string {
	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return strings.Join(ids, ", ")
}

func describeCommentAddedEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	author, body, ok := decodeCommentEvent(event)
	if !ok {
		return describeUnknownEvent(event)
	}
	return fmt.Sprintf("%s commented: %s", author, body)
}

func describeArtifactAddedEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	data, err := decodeEventPayload[orderArtifactAddedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	description := "artifact"
	if data.Artifact != nil {
		description = data.Artifact.Type + " artifact"
		if data.Artifact.ID != "" {
			description += " " + data.Artifact.ID
		}
	}

	line := "added " + description
	if actor := describeEventActor(data.User, data.Automation, nil); actor != "" {
		line += " by " + actor
	}
	return line
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
