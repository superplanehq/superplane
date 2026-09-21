package jira

import (
	"net/url"
	"strings"
)

// IssueKeyFromEventData reads the Jira issue key from a canvas root event
// envelope. The envelope looks like:
//
//	{ "type": "jira.issue", "data": { "issue": { "key": "ENG-5" } } }
func IssueKeyFromEventData(eventData any) (string, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return "", false
	}
	if typeName, _ := envelope["type"].(string); typeName != IssueEventPayloadType {
		return "", false
	}

	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return "", false
	}
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return "", false
	}
	issueKey := strings.TrimSpace(issueKeyString(issue["key"]))
	if issueKey == "" {
		return "", false
	}
	return issueKey, true
}

// IssueURLFragment is the origin-URL substring that identifies a Jira issue.
// Callers use it for a coarse lookup, then confirm with IssueKeyFromURL so
// `/browse/ENG-5` cannot match `/browse/ENG-50`.
func IssueURLFragment(issueKey string) string {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return ""
	}
	return "/browse/" + issueKey
}

// IssueKeyFromURL reads the Jira issue key from an issue page URL.
// It accepts `/browse/<key>` with or without a trailing slash or query
// string. Other paths are ignored.
func IssueKeyFromURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", false
	}
	if parsed.Hostname() == "" {
		return "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "browse") {
		return "", false
	}

	issueKey := strings.TrimSpace(parts[1])
	if issueKey == "" {
		return "", false
	}
	return issueKey, true
}

func issueKeyString(value any) string {
	key, _ := value.(string)
	return key
}
