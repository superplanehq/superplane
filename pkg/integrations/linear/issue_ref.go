package linear

import (
	"net/url"
	"strings"
)

// IssueRef identifies one Linear issue by its browse URL.
type IssueRef struct {
	ID         string
	Identifier string
	Host       string
}

func (r IssueRef) valid() bool {
	return r.Identifier != "" && r.Host != ""
}

// IssueRefFromEventData reads the Linear issue from a canvas root event
// envelope. The envelope looks like:
//
//	{ "type": "linear.issue", "data": { "url": "https://linear.app/acme/issue/ENG-142/slug", "data": { "id": "...", "identifier": "ENG-142" } } }
func IssueRefFromEventData(eventData any) (IssueRef, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return IssueRef{}, false
	}
	if typeName, _ := envelope["type"].(string); typeName != IssuePayloadType {
		return IssueRef{}, false
	}

	body, ok := envelope["data"].(map[string]any)
	if !ok {
		return IssueRef{}, false
	}
	rawURL, _ := body["url"].(string)
	ref, ok := IssueRefFromURL(rawURL)
	if !ok {
		return IssueRef{}, false
	}

	issue, _ := body["data"].(map[string]any)
	if id, _ := issue["id"].(string); strings.TrimSpace(id) != "" {
		ref.ID = strings.TrimSpace(id)
	}
	if identifier, _ := issue["identifier"].(string); strings.TrimSpace(identifier) != "" {
		ref.Identifier = strings.TrimSpace(identifier)
	}
	return ref, ref.valid()
}

// IssueIDFromEventData reads the Linear issue id used by the API.
func IssueIDFromEventData(eventData any) (string, bool) {
	ref, ok := IssueRefFromEventData(eventData)
	if !ok || ref.ID == "" {
		return "", false
	}
	return ref.ID, true
}

// IssueURLFragment is the origin-URL substring that identifies a Linear
// issue. Callers use it for a coarse lookup, then confirm with
// IssueRefFromURL so ENG-142 cannot match ENG-1420.
func IssueURLFragment(ref IssueRef) string {
	if !ref.valid() {
		return ""
	}
	return ref.Host + "/issue/" + ref.Identifier
}

// IssueRefFromURL reads the issue identifier from a Linear issue page URL.
// The path is /{workspace}/issue/{identifier} with an optional slug.
func IssueRefFromURL(rawURL string) (IssueRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return IssueRef{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 3 || !strings.EqualFold(parts[1], "issue") {
		return IssueRef{}, false
	}
	identifier := strings.TrimSpace(parts[2])
	host := strings.ToLower(parsed.Hostname())
	if identifier == "" || host == "" {
		return IssueRef{}, false
	}
	return IssueRef{Identifier: identifier, Host: host}, true
}
