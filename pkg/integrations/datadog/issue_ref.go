package datadog

import (
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// IssueIDFromURL reads the Error Tracking issue id from a Datadog issue page.
func IssueIDFromURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return "", false
	}

	match := issuePathMarker.FindStringSubmatch(parsed.Path)
	if len(match) < 2 {
		return "", false
	}
	return strings.ToLower(match[1]), true
}

// IssueIDFromEventData reads the Error Tracking issue id from a canvas root
// event envelope. Seeded issues carry the issue page in link. A monitor
// alert link does not.
func IssueIDFromEventData(eventData any) (string, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return "", false
	}
	if typeName, _ := envelope["type"].(string); typeName != ErrorTrackingAlertPayloadType {
		return "", false
	}

	payload, ok := envelope["data"].(map[string]any)
	if !ok {
		return "", false
	}
	link, _ := payload["link"].(string)
	return IssueIDFromURL(link)
}

// IssueHasWorkOrder reports whether this factory already has a work order
// for the Error Tracking issue. Matching covers every work-order state.
func IssueHasWorkOrder(tx *gorm.DB, factory *models.Factory, issueID string) (bool, error) {
	issueID = strings.ToLower(strings.TrimSpace(issueID))
	if factory == nil || issueID == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, "/error-tracking/issue/"+issueID)
	if err != nil {
		return false, err
	}

	for _, rawURL := range urls {
		found, ok := IssueIDFromURL(rawURL)
		if ok && found == issueID {
			return true, nil
		}
	}

	return false, nil
}
