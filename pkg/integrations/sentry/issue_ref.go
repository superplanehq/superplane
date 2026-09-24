package sentry

import (
	"encoding/json"
	"fmt"
	"net/url"
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

// IssueURLFragment is the origin-URL substring that identifies a Sentry
// issue. Callers use it for a coarse lookup, then confirm with
// IssueIDFromURL so `/issues/12` cannot match `/issues/123`.
func IssueURLFragment(issueID string) string {
	issueID = strings.TrimSpace(issueID)
	if issueID == "" {
		return ""
	}
	return "/issues/" + issueID
}

// IssueIDFromURL reads the numeric Sentry issue ID from an issue page URL.
// It accepts `/issues/<id>/` and `/organizations/<org>/issues/<id>/` on
// sentry.io and on self-hosted instances, with or without a trailing slash
// or query string. GitHub-style `/owner/repo/issues/<id>` paths are ignored.
func IssueIDFromURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", false
	}
	if parsed.Hostname() == "" {
		return "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if !isSentryIssuePath(parts) {
		return "", false
	}

	for i, part := range parts {
		if part != "issues" || i+1 >= len(parts) {
			continue
		}
		issueID := strings.TrimSpace(parts[i+1])
		if !isNumericIssueID(issueID) {
			return "", false
		}
		return issueID, true
	}

	return "", false
}

func isSentryIssuePath(parts []string) bool {
	if len(parts) >= 2 && parts[0] == "issues" {
		return true
	}
	if len(parts) < 4 || parts[0] != "organizations" {
		return false
	}
	for i := 1; i < len(parts)-1; i++ {
		if parts[i] == "issues" {
			return true
		}
	}
	return false
}

func isNumericIssueID(issueID string) bool {
	if issueID == "" {
		return false
	}
	for _, r := range issueID {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
