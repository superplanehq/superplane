package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	eventTypeTaskAssigneesUpdated  = "order.assignees.updated"
	eventTypeTaskStatusUpdated     = "order.status.updated"
	eventTypeTaskCommentAdded      = "order.comment.added"
	eventTypeTaskArtifactAdded     = "order.artifact.added"
	eventTypeStepExecutionCreated  = "step.execution.created"
	eventTypeStepExecutionFinished = "step.execution.finished"
)

func formatTaskState(state openapi_client.FactoriesWorkOrderState) string {
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

func formatTaskResult(result openapi_client.FactoriesWorkOrderResult) string {
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

func formatTaskCreator(creator openapi_client.FactoriesWorkOrderCreator) string {
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

const unknownActorLabel = "unknown user"

type memberEmailLookup struct {
	emailByID map[string]string
}

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

func (l memberEmailLookup) actorLabel(userID string) string {
	if userID == "" {
		return unknownActorLabel
	}
	if label, ok := l.emailByID[userID]; ok && label != "" {
		return label
	}
	return unknownActorLabel
}

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
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

type taskStatusUpdatedEvent struct {
	User       *eventUserRef       `json:"user,omitempty"`
	Automation *eventAutomationRef `json:"automation,omitempty"`
	Run        *eventRunRef        `json:"run,omitempty"`
	FromState  string              `json:"fromState"`
	ToState    string              `json:"toState"`
	FromResult string              `json:"fromResult,omitempty"`
	ToResult   string              `json:"toResult,omitempty"`
}

type taskAssigneesUpdatedEvent struct {
	User       *eventUserRef  `json:"user,omitempty"`
	Assigned   []eventUserRef `json:"assigned,omitempty"`
	Unassigned []eventUserRef `json:"unassigned,omitempty"`
}

type commentAuthorEvent struct {
	Kind       string              `json:"kind"`
	UserID     string              `json:"userId,omitempty"`
	Automation *eventAutomationRef `json:"automation,omitempty"`
}

type taskCommentAddedEvent struct {
	Body   string              `json:"body"`
	Author *commentAuthorEvent `json:"author,omitempty"`
}

type taskArtifactAddedEvent struct {
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

func decodeEventPayload[T any](payload map[string]any) (*T, error) {
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

func decodeCommentEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) (author string, body string, ok bool) {
	data, err := decodeEventPayload[taskCommentAddedEvent](event.GetEvent())
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

func describeEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) string {
	switch event.GetType() {
	case eventTypeTaskStatusUpdated:
		return describeStatusUpdatedEvent(event, lookup)
	case eventTypeTaskAssigneesUpdated:
		return describeAssigneesUpdatedEvent(event, lookup)
	case eventTypeTaskCommentAdded:
		return describeCommentAddedEvent(event, lookup)
	case eventTypeTaskArtifactAdded:
		return describeArtifactAddedEvent(event)
	case eventTypeStepExecutionCreated:
		return describeStepExecutionEvent(event, "started")
	case eventTypeStepExecutionFinished:
		return describeStepExecutionEvent(event, "finished")
	default:
		return describeUnknownEvent(event)
	}
}

func describeStatusTransition(fromState, toState, toResult string) string {
	switch {
	case fromState == "":
		return "Task created"
	case toState == "closed":
		if toResult != "" {
			return fmt.Sprintf("Task closed as %s", titleCase(toResult))
		}
		return "Task closed"
	case fromState == "draft" && toState == "open":
		return "Task opened"
	case fromState == "open" && toState == "draft":
		return "Task moved back to Draft"
	case fromState == "closed" && toState == "open":
		return "Task reopened"
	default:
		return fmt.Sprintf("Task moved from %s to %s", titleCase(fromState), titleCase(toState))
	}
}

func describeStatusUpdatedEvent(event openapi_client.FactoriesWorkOrderEvent, lookup memberEmailLookup) string {
	data, err := decodeEventPayload[taskStatusUpdatedEvent](event.GetEvent())
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
	data, err := decodeEventPayload[taskAssigneesUpdatedEvent](event.GetEvent())
	if err != nil {
		return describeUnknownEvent(event)
	}

	var parts []string
	if len(data.Assigned) > 0 {
		parts = append(parts, "Task assigned to "+joinEventActors(data.Assigned, lookup))
	}
	if len(data.Unassigned) > 0 {
		parts = append(parts, "unassigned "+joinEventActors(data.Unassigned, lookup))
	}
	if len(parts) == 0 {
		parts = append(parts, "Task assignees updated")
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

func artifactTypeName(t string) string {
	switch t {
	case "pr":
		return "PR"
	default:
		return t
	}
}

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

func describeArtifactAddedEvent(event openapi_client.FactoriesWorkOrderEvent) string {
	data, err := decodeEventPayload[taskArtifactAddedEvent](event.GetEvent())
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
