package productive

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// BaseURL is Productive.io's API v2 base URL, used unless the "region"
	// configuration field overrides it.
	BaseURL = "https://api.productive.io/api/v2"

	// AuthTokenHeader carries the Productive.io API token on every request.
	AuthTokenHeader = "X-Auth-Token"

	// OrganizationIDHeader scopes every request to one Productive.io
	// organization. Productive.io issues tokens per person, not per
	// organization, so this header is required on every call.
	OrganizationIDHeader = "X-Organization-Id"

	// TaskPayloadType is the payload type the onTask trigger emits.
	TaskPayloadType = "productive.task"

	// ResourceTypeProject is the resource type projects are listed and picked
	// as, both for the onTask trigger's project field and ListResources.
	ResourceTypeProject = "project"

	// TaskCreatedEvent and TaskUpdatedEvent name the change a task event
	// carries, in the "meta" object of the emitted envelope.
	TaskCreatedEvent = "task.created"
	TaskUpdatedEvent = "task.updated"

	// ActionCreated and ActionUpdated are the values of the onTask trigger's
	// "actions" field.
	ActionCreated = "created"
	ActionUpdated = "updated"

	// WebhookTypeStandard is Productive.io type_id for a normal webhook
	// (not Zapier).
	WebhookTypeStandard = 1

	// EventNewTask and EventUpdatedTask are Productive.io event_id values
	// for task created and task updated.
	EventNewTask     = 1
	EventUpdatedTask = 24

	// TaskTypeRegular is a normal Productive.io task. TaskTypeMilestone is
	// a key task.
	TaskTypeRegular   = 1
	TaskTypeMilestone = 3

	// EventHeader is a custom header SuperPlane asks Productive.io to send.
	// Productive.io does not send it on real deliveries, so the signature
	// token is what tells a created task from an updated task.
	EventHeader = "X-Productive-Event"

	// SignatureHeader carries Productive.io's HMAC of timestamp + "." + body.
	SignatureHeader = "Productive-Signature"
)

// NodeMetadata is stored on productive.onTask nodes, so canvas cards can show
// the project without re-querying Productive.io.
type NodeMetadata struct {
	Project *Project `json:"project,omitempty" mapstructure:"project,omitempty"`
}

// TaskEnvelope wraps a task resource the way every consumer of this trigger
// reads it: the JSON:API resource under "data", and the change that produced
// it under "meta". Seeded tasks use the same shape, so nothing downstream can
// tell a seeded task from a polled one.
func TaskEnvelope(event string, document map[string]any) map[string]any {
	return map[string]any{
		"meta": map[string]any{"event": event},
		"data": document,
	}
}

// actionForEvent maps the event a webhook delivery reports back to the value
// the onTask trigger's "actions" field uses, so a delivery can be checked
// against what a node was configured to listen for.
func actionForEvent(event string) (string, bool) {
	switch event {
	case TaskCreatedEvent:
		return ActionCreated, true
	case TaskUpdatedEvent:
		return ActionUpdated, true
	default:
		return "", false
	}
}

// webhookSecretEntry is one Productive.io signature token and the event
// that webhook was registered for. A delivery is signed with exactly one
// of these tokens, which is how SuperPlane tells created from updated.
type webhookSecretEntry struct {
	event string
	token string
}

// signedWebhookEvent checks Productive-Signature (t=<unix>, s=<hex>)
// against HMAC-SHA256 of timestamp + "." + raw body. When one stored
// token matches, it returns that token's event. When several tokens
// match, the event is empty and the caller may use EventHeader.
func signedWebhookEvent(ctx core.WebhookRequestContext) (string, int, error) {
	timestamp, signature, ok := parseProductiveSignature(ctx.Headers.Get(SignatureHeader))
	if !ok {
		return "", http.StatusForbidden, fmt.Errorf("missing %s header", SignatureHeader)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	entries := webhookSecretEntries(secret)
	if len(entries) == 0 {
		return "", http.StatusInternalServerError, fmt.Errorf("missing webhook secret")
	}

	matched := []string{}
	seen := map[string]bool{}
	for _, entry := range entries {
		expected := webhookSignatureHex([]byte(entry.token), timestamp, ctx.Body)
		if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
			continue
		}
		if seen[entry.event] {
			continue
		}
		seen[entry.event] = true
		matched = append(matched, entry.event)
	}

	if len(matched) == 0 {
		return "", http.StatusForbidden, fmt.Errorf("invalid webhook signature")
	}
	if len(matched) == 1 {
		return matched[0], http.StatusOK, nil
	}
	return "", http.StatusOK, nil
}

// webhookSecretEntries reads the signature tokens stored at registration.
// New records are "event=token" lines. Older records with two lines are
// created then updated. One older line is a token both webhooks shared,
// so it does not name the event.
func webhookSecretEntries(secret []byte) []webhookSecretEntry {
	lines := []string{}
	for _, part := range strings.Split(string(secret), "\n") {
		part = strings.TrimSpace(part)
		if part != "" {
			lines = append(lines, part)
		}
	}

	labeled := false
	for _, line := range lines {
		if strings.Contains(line, "=") {
			labeled = true
			break
		}
	}
	if !labeled {
		return legacyWebhookSecretEntries(lines)
	}

	entries := make([]webhookSecretEntry, 0, len(lines))
	for _, line := range lines {
		event, token, ok := strings.Cut(line, "=")
		event = strings.TrimSpace(event)
		token = strings.TrimSpace(token)
		if !ok || event == "" || token == "" {
			continue
		}
		entries = append(entries, webhookSecretEntry{event: event, token: token})
	}
	return entries
}

func legacyWebhookSecretEntries(tokens []string) []webhookSecretEntry {
	// The previous registration stored one copy of a token that both
	// webhooks shared. That copy does not say which event arrived.
	if len(tokens) < 2 {
		if len(tokens) == 0 {
			return nil
		}
		return []webhookSecretEntry{{token: tokens[0]}}
	}

	entries := make([]webhookSecretEntry, 0, len(tokens))
	for i, token := range tokens {
		if i >= len(remoteWebhookEvents) {
			break
		}
		entries = append(entries, webhookSecretEntry{
			event: remoteWebhookEvents[i].eventName,
			token: token,
		})
	}
	return entries
}

func webhookSignatureHex(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// parseProductiveSignature reads t= and s= from a Productive-Signature header.
func parseProductiveSignature(header string) (timestamp, signature string, ok bool) {
	for _, part := range strings.Split(header, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "t":
			timestamp = strings.TrimSpace(value)
		case "s":
			signature = strings.TrimSpace(value)
		}
	}
	return timestamp, signature, timestamp != "" && signature != ""
}

// taskProjectID reads the project relationship id from a JSON:API task.
func taskProjectID(document map[string]any) string {
	relationships, _ := document["relationships"].(map[string]any)
	project, _ := relationships["project"].(map[string]any)
	data, _ := project["data"].(map[string]any)
	id, _ := data["id"].(string)
	return strings.TrimSpace(id)
}

// taskTypeID reads attributes.type_id from a JSON:API task.
func taskTypeID(document map[string]any) int {
	attributes, _ := document["attributes"].(map[string]any)
	return numberAttribute(attributes["type_id"])
}

func numberAttribute(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
