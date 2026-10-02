package linear

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// ReceiptField is the canvas payload key for the stored receipt ID.
// Linear does not send this field.
const ReceiptField = "superplaneReceiptId"

// WebhookSummary is the metadata stored for one Linear call.
// It does not include the body, the signature, or actor details.
type WebhookSummary struct {
	EventType       string
	Action          string
	IssueIdentifier string
	IssueID         string
	TeamKey         string
	WorkspaceKey    string
}

// SummarizeWebhook reads event metadata from a Linear webhook body.
func SummarizeWebhook(headers http.Header, body []byte) WebhookSummary {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		payload = map[string]any{}
	}

	eventType := ""
	if headers != nil {
		eventType = strings.TrimSpace(headers.Get(EventHeader))
	}
	if eventType == "" {
		eventType = webhookString(payload, "type")
	}

	data, _ := payload["data"].(map[string]any)
	issue := linearIssueFields(eventType, data)
	return WebhookSummary{
		EventType:       eventType,
		Action:          webhookString(payload, "action"),
		IssueIdentifier: issue.identifier,
		IssueID:         issue.id,
		TeamKey:         issue.teamKey,
		WorkspaceKey:    workspaceKey(webhookString(payload, "url"), webhookString(data, "url")),
	}
}

type linearIssueFieldsResult struct {
	id         string
	identifier string
	teamKey    string
}

func linearIssueFields(eventType string, data map[string]any) linearIssueFieldsResult {
	if data == nil {
		return linearIssueFieldsResult{}
	}

	if !strings.EqualFold(eventType, IssueResourceType) {
		if issue, ok := data["issue"].(map[string]any); ok {
			return linearIssueFieldsResult{
				id:         webhookString(issue, "id"),
				identifier: webhookString(issue, "identifier"),
				teamKey:    teamKey(issue),
			}
		}
		if issueID := webhookString(data, "issueId"); issueID != "" {
			return linearIssueFieldsResult{id: issueID}
		}
	}

	return linearIssueFieldsResult{
		id:         webhookString(data, "id"),
		identifier: webhookString(data, "identifier"),
		teamKey:    teamKey(data),
	}
}

func teamKey(data map[string]any) string {
	team, _ := data["team"].(map[string]any)
	return webhookString(team, "key")
}

func workspaceKey(urls ...string) string {
	for _, raw := range urls {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed.Host == "" {
			continue
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		return parts[0]
	}
	return ""
}

func webhookString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

// ReceiptIDFromEventData reads the SuperPlane webhook receipt ID stored on a
// Linear canvas event. Linear does not send this field.
func ReceiptIDFromEventData(eventData any) (uuid.UUID, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return uuid.Nil, false
	}
	if typeName, _ := envelope["type"].(string); typeName != "" && !strings.HasPrefix(typeName, "linear.") {
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
