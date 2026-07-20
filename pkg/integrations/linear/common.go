package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// IssuePayloadType is the payload type emitted for every Linear issue,
	// both by the createIssue action and the onIssue trigger.
	IssuePayloadType = "linear.issue"

	// SignatureHeader carries a hex-encoded HMAC-SHA256 of the raw request body.
	SignatureHeader = "Linear-Signature"

	// EventHeader carries the resource type that triggered the delivery, e.g. "Issue".
	EventHeader = "Linear-Event"

	// IssueResourceType is the Linear webhook resource type for issue events.
	IssueResourceType = "Issue"
)

// NodeMetadata is stored on Linear nodes at setup time, so canvas cards can
// show the team without re-querying Linear.
type NodeMetadata struct {
	Team *Team `json:"team,omitempty" mapstructure:"team,omitempty"`
}

// requireTeam resolves a team ID against the integration metadata populated
// during sync, so setup fails fast on a team the API key cannot reach.
func requireTeam(integration core.IntegrationContext, teamID string) (*Team, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	for _, team := range metadata.Teams {
		if team.ID == teamID {
			t := team
			return &t, nil
		}
	}

	return nil, fmt.Errorf("team %s not found", teamID)
}

// verifyWebhookSignature checks the Linear-Signature header against an HMAC-SHA256
// of the raw request body. Linear signs the bytes exactly as delivered, so the
// raw body must be used rather than a re-serialized payload.
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
