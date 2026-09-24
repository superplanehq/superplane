package jira

import (
	"net/url"
	"strings"
)

// IssueRef identifies one Jira issue on one site.
type IssueRef struct {
	Key  string
	Host string
}

func (r IssueRef) valid() bool {
	return r.Key != "" && r.Host != ""
}

func newIssueRef(host, key string) (IssueRef, bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	key = strings.TrimSpace(key)
	if host == "" || key == "" {
		return IssueRef{}, false
	}
	return IssueRef{Host: host, Key: key}, true
}

// IssueRefFromEventData reads the Jira issue key and site host from a canvas
// root event envelope. The envelope looks like:
//
//	{ "type": "jira.issue", "data": { "url": "https://acme.atlassian.net/browse/ENG-5", "issue": { "key": "ENG-5" } } }
func IssueRefFromEventData(eventData any) (IssueRef, bool) {
	issueKey, ok := IssueKeyFromEventData(eventData)
	if !ok {
		return IssueRef{}, false
	}

	envelope, ok := eventData.(map[string]any)
	if !ok {
		return IssueRef{}, false
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return IssueRef{}, false
	}
	rawURL, _ := data["url"].(string)
	fromURL, ok := IssueRefFromURL(rawURL)
	if !ok {
		return IssueRef{}, false
	}
	return newIssueRef(fromURL.Host, issueKey)
}

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

// IssueRefFromSite builds an issue identity from a site address and issue key.
func IssueRefFromSite(siteURL, issueKey string) (IssueRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil {
		return IssueRef{}, false
	}
	return newIssueRef(parsed.Hostname(), issueKey)
}

// IssueURLFragment is the origin-URL substring that identifies a Jira issue
// on one site. Callers use it for a coarse lookup, then confirm with
// IssueRefFromURL so `acme.../browse/ENG-5` cannot match `acme.../browse/ENG-50`
// or the same key on another host.
func IssueURLFragment(ref IssueRef) string {
	if !ref.valid() {
		return ""
	}
	return ref.Host + "/browse/" + ref.Key
}

// IssueRefFromURL reads the Jira issue key and site host from an issue page
// URL. It accepts `/browse/<key>` with or without a trailing slash or query
// string. Other paths are ignored.
func IssueRefFromURL(rawURL string) (IssueRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return IssueRef{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "browse") {
		return IssueRef{}, false
	}

	return newIssueRef(parsed.Hostname(), parts[1])
}

// IssueKeyFromURL reads the Jira issue key from an issue page URL.
func IssueKeyFromURL(rawURL string) (string, bool) {
	ref, ok := IssueRefFromURL(rawURL)
	return ref.Key, ok
}

func issueKeyString(value any) string {
	key, _ := value.(string)
	return key
}
