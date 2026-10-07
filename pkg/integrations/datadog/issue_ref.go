package datadog

import (
	"net/url"
	"strings"

	"github.com/google/uuid"
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
	if issueID, ok := IssueIDFromURL(link); ok {
		return issueID, true
	}
	return issueIDInValue(payload)
}

// ReceiptIDFromEventData reads the SuperPlane webhook receipt ID stored on a
// datadog.errorTrackingAlert canvas event. Datadog does not send this field.
func ReceiptIDFromEventData(eventData any) (uuid.UUID, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return uuid.Nil, false
	}
	if typeName, _ := envelope["type"].(string); typeName != "" && typeName != ErrorTrackingAlertPayloadType {
		return uuid.Nil, false
	}

	payload, ok := envelope["data"].(map[string]any)
	if !ok {
		return uuid.Nil, false
	}
	raw, _ := payload[ReceiptField].(string)
	receiptID, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, false
	}
	return receiptID, true
}

func issueIDInValue(value any) (string, bool) {
	switch current := value.(type) {
	case string:
		if issueID, ok := IssueIDFromURL(current); ok {
			return issueID, true
		}
		match := issuePathMarker.FindStringSubmatch(current)
		if len(match) < 2 {
			return "", false
		}
		return strings.ToLower(match[1]), true
	case map[string]any:
		for _, child := range current {
			if issueID, ok := issueIDInValue(child); ok {
				return issueID, true
			}
		}
	case []any:
		for _, child := range current {
			if issueID, ok := issueIDInValue(child); ok {
				return issueID, true
			}
		}
	}
	return "", false
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
