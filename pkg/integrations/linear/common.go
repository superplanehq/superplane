package linear

import (
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

	// CommentPayloadType is the payload type emitted by the addIssueComment action.
	CommentPayloadType = "linear.comment"

	// AttachmentPayloadType is the payload type emitted for every Linear attachment,
	// both by the createAttachment action and the onIssueAttachment trigger.
	AttachmentPayloadType = "linear.attachment"

	// ReactionPayloadType is the payload type emitted by the addReaction action.
	ReactionPayloadType = "linear.reaction"

	// SignatureHeader carries a hex-encoded HMAC-SHA256 of the raw request body.
	SignatureHeader = "Linear-Signature"

	// EventHeader carries the resource type that triggered the delivery, e.g. "Issue".
	EventHeader = "Linear-Event"

	// IssueResourceType is the Linear webhook resource type for issue events.
	IssueResourceType = "Issue"

	// CommentResourceType is the Linear webhook resource type for comment events.
	CommentResourceType = "Comment"

	// AttachmentResourceType is the Linear webhook resource type for attachment events.
	AttachmentResourceType = "Attachment"
)

const triggerWebhookDocumentation = `## Webhook Setup

SuperPlane receives Linear events for this trigger. When the Linear OAuth application has a webhook URL and a signing secret, SuperPlane uses that webhook. The connection needs the **read** and **write** scopes.

When the application has no signing secret, SuperPlane creates a webhook in Linear for this trigger and removes it when the trigger is deleted. Linear allows that only for a workspace admin or a token with the **admin** scope.`

// NodeMetadata is stored on Linear nodes at setup time, so canvas cards can
// show the team without re-querying Linear.
type NodeMetadata struct {
	Team *Team `json:"team,omitempty" mapstructure:"team,omitempty"`
}

// requireTeam resolves a team ID against the integration metadata populated
// during sync, so setup fails fast on a team the connection cannot reach.
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
// An application webhook is signed with the OAuth application's signing secret.
// A webhook created through the API is signed with the secret stored on that webhook.
func verifyWebhookSignature(ctx core.WebhookRequestContext) (int, error) {
	signature := strings.TrimSpace(ctx.Headers.Get(SignatureHeader))
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing %s header", SignatureHeader)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil && AppWebhookSigningSecret(ctx.Integration) == "" {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if SignatureMatches(signature, ctx.Body, secret) {
		return http.StatusOK, nil
	}

	appSecret := AppWebhookSigningSecret(ctx.Integration)
	if SignatureMatches(signature, ctx.Body, []byte(appSecret)) {
		return http.StatusOK, nil
	}

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}
	if len(secret) == 0 && appSecret == "" {
		return http.StatusInternalServerError, fmt.Errorf("missing webhook secret")
	}

	return http.StatusForbidden, fmt.Errorf("invalid webhook signature")
}
