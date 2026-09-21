package sentry

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IssuePayloadType is the canvas event type emitted by sentry.onIssue and
// by Sentry intake seeding.
const IssuePayloadType = "sentry.issue"

const (
	IssueStatusUnresolved            = "unresolved"
	IssueStatusResolved              = "resolved"
	IssueStatusResolvedInNextRelease = "resolvedInNextRelease"
	IssueStatusIgnored               = "ignored"
)

// IssueIDFromEventData reads the Sentry numeric issue ID from a canvas root
// event envelope. The envelope looks like:
//
//	{ "type": "sentry.issue", "data": { "data": { "issue": { "id": "123" } } } }
func IssueIDFromEventData(eventData any) (string, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return "", false
	}
	if typeName, _ := envelope["type"].(string); typeName != IssuePayloadType {
		return "", false
	}

	webhook, ok := envelope["data"].(map[string]any)
	if !ok {
		return "", false
	}
	data, ok := webhook["data"].(map[string]any)
	if !ok {
		return "", false
	}
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return "", false
	}
	issueID := issueIDString(issue["id"])
	if issueID == "" {
		return "", false
	}
	return issueID, true
}

// IssueStatusIsSettled reports whether SuperPlane should leave the Sentry
// issue alone. A completed task only needs to resolve an open issue.
func IssueStatusIsSettled(status string) bool {
	switch strings.TrimSpace(status) {
	case IssueStatusResolved, IssueStatusResolvedInNextRelease, IssueStatusIgnored:
		return true
	default:
		return false
	}
}

func issueIDString(value any) string {
	switch current := value.(type) {
	case string:
		return strings.TrimSpace(current)
	case json.Number:
		return strings.TrimSpace(current.String())
	case int:
		return fmt.Sprintf("%d", current)
	case int64:
		return fmt.Sprintf("%d", current)
	case float64:
		return fmt.Sprintf("%.0f", current)
	default:
		return ""
	}
}
