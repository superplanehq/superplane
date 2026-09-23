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
	// carries, in the "meta" object of the emitted envelope. SuperPlane
	// asks Productive.io to send the same names in EventHeader.
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

	// EventHeader is set as a Productive.io custom header on each remote
	// webhook so a shared SuperPlane URL can tell created from updated.
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

// verifyWebhookSignature checks Productive-Signature (t=<unix>, s=<hex>)
// against HMAC-SHA256 of timestamp + "." + raw body, keyed with the
// signature token Productive.io returned at registration.
func verifyWebhookSignature(ctx core.WebhookRequestContext) (int, error) {
	timestamp, signature, ok := parseProductiveSignature(ctx.Headers.Get(SignatureHeader))
	if !ok {
		return http.StatusForbidden, fmt.Errorf("missing %s header", SignatureHeader)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if len(secret) == 0 {
		return http.StatusInternalServerError, fmt.Errorf("missing webhook secret")
	}

	for _, token := range webhookSecretTokens(secret) {
		if hmac.Equal([]byte(strings.ToLower(signature)), []byte(webhookSignatureHex(token, timestamp, ctx.Body))) {
			return http.StatusOK, nil
		}
	}

	return http.StatusForbidden, fmt.Errorf("invalid webhook signature")
}

func webhookSecretTokens(secret []byte) [][]byte {
	parts := strings.Split(string(secret), "\n")
	tokens := make([][]byte, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		tokens = append(tokens, []byte(token))
	}
	return tokens
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
