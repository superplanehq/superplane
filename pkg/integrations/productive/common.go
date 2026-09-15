package productive

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
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
	// carries, in the "meta" object of the emitted envelope. Productive.io
	// sends the same names in the EventHeader of a webhook delivery.
	TaskCreatedEvent = "task.created"
	TaskUpdatedEvent = "task.updated"

	// ActionCreated and ActionUpdated are the values of the onTask trigger's
	// "actions" field.
	ActionCreated = "created"
	ActionUpdated = "updated"

	// EventHeader carries the event a webhook delivery reports, one of
	// TaskCreatedEvent or TaskUpdatedEvent.
	EventHeader = "X-Productive-Event"

	// SignatureHeader carries a hex-encoded HMAC-SHA256 of the raw request
	// body, keyed with the webhook's secret.
	SignatureHeader = "X-Productive-Signature"
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

// verifyWebhookSignature checks the SignatureHeader against an HMAC-SHA256 of
// the raw request body, keyed with the node's webhook secret. Productive.io
// signs the bytes exactly as delivered, so the raw body must be used rather
// than a re-serialized payload.
func verifyWebhookSignature(ctx core.WebhookRequestContext) (int, error) {
	signature := strings.TrimSpace(ctx.Headers.Get(SignatureHeader))
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing %s header", SignatureHeader)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if len(secret) == 0 {
		return http.StatusInternalServerError, fmt.Errorf("missing webhook secret")
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(ctx.Body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
		return http.StatusForbidden, fmt.Errorf("invalid webhook signature")
	}

	return http.StatusOK, nil
}
