package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const appWebhookPath = "/api/v1/linear/webhook"

// appWebhookResourceTypes are the Linear resources SuperPlane triggers listen to.
// The label trigger reads Issue events and filters label changes.
var appWebhookResourceTypes = []string{
	IssueResourceType,
	CommentResourceType,
	AttachmentResourceType,
}

// AppEvent is the part of a Linear webhook SuperPlane uses to choose a subscription.
type AppEvent struct {
	OrganizationID string
	WorkspaceKey   string
	ResourceType   string
	TeamID         string
	TeamKey        string
}

// AppWebhookURL is the address to enter on a Linear OAuth application.
func AppWebhookURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + appWebhookPath
}

// OAuthScopes is the scope list for the authorize URL.
// The admin scope is omitted when this connection can verify the application webhook.
func OAuthScopes(integration core.IntegrationContext) string {
	if AppWebhookSigningSecret(integration) != "" {
		return scopeReadWrite
	}
	return scopeWithAdmin
}

// AppWebhookSigningSecret is the secret Linear uses to sign the application webhook.
// A customer application stores the secret on the connection. SuperPlane's application
// stores it in the process environment.
func AppWebhookSigningSecret(integration core.IntegrationContext) string {
	if integration == nil {
		return HostedWebhookSecret()
	}

	app := resolveOAuthApp(integration)
	if !app.Hosted {
		secret, err := integration.GetConfig(WebhookSecretConfig)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(secret))
	}

	return HostedWebhookSecret()
}

// SignatureMatches reports whether signature is the hex HMAC-SHA256 of body.
func SignatureMatches(signature string, body, secret []byte) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" || len(secret) == 0 || len(body) == 0 {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected))
}

// ParseAppEvent reads the routing fields from a Linear webhook.
func ParseAppEvent(headers http.Header, body []byte) (AppEvent, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return AppEvent{}, err
	}

	eventType := ""
	if headers != nil {
		eventType = strings.TrimSpace(headers.Get(EventHeader))
	}
	if eventType == "" {
		eventType = webhookString(payload, "type")
	}

	data, _ := payload["data"].(map[string]any)
	return AppEvent{
		OrganizationID: webhookString(payload, "organizationId"),
		WorkspaceKey:   workspaceKey(webhookString(payload, "url"), webhookString(data, "url")),
		ResourceType:   eventType,
		TeamID:         eventTeamID(data),
		TeamKey:        eventTeamKey(data),
	}, nil
}

// IsAppLevelWebhook reports whether this subscription receives the application webhook.
func IsAppLevelWebhook(metadata any) bool {
	decoded := WebhookMetadata{}
	if err := mapstructure.Decode(metadata, &decoded); err != nil {
		return false
	}
	return decoded.AppLevel
}

// SubscriptionMatches reports whether a local subscription wants this Linear event.
// A subscription with no team listens to every team. A missing team on the event
// does not match a team-scoped subscription.
func SubscriptionMatches(config WebhookConfiguration, event AppEvent, teams []Team) bool {
	if config.ResourceType != "" && !strings.EqualFold(config.ResourceType, event.ResourceType) {
		return false
	}

	teamIDs := config.resolvedTeamIDs()
	if len(teamIDs) == 0 {
		return true
	}

	teamID := event.TeamID
	if teamID == "" {
		teamID = teamIDForKey(teams, event.TeamKey)
	}
	if teamID == "" {
		return false
	}

	return slices.Contains(teamIDs, teamID)
}

func eventTeamID(data map[string]any) string {
	if teamID := teamIDOf(data); teamID != "" {
		return teamID
	}
	issue, _ := data["issue"].(map[string]any)
	return teamIDOf(issue)
}

func teamIDOf(data map[string]any) string {
	if data == nil {
		return ""
	}
	if teamID := webhookString(data, "teamId"); teamID != "" {
		return teamID
	}
	team, _ := data["team"].(map[string]any)
	return webhookString(team, "id")
}

func eventTeamKey(data map[string]any) string {
	if data == nil {
		return ""
	}
	if key := teamKey(data); key != "" {
		return key
	}
	issue, _ := data["issue"].(map[string]any)
	return teamKey(issue)
}

func teamIDForKey(teams []Team, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	for _, team := range teams {
		if team.Key == key {
			return team.ID
		}
	}
	return ""
}
