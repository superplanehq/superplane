package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
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
	// IssueID is the parent issue when the payload is not itself an issue.
	IssueID string
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

// scopeIncludesAdmin reports whether scopes contains the admin scope.
func scopeIncludesAdmin(scopes string) bool {
	for _, scope := range strings.Split(scopes, ",") {
		if strings.EqualFold(strings.TrimSpace(scope), "admin") {
			return true
		}
	}
	return false
}

// DeliveryKey identifies one Linear payload. Linear retries send the same body,
// so a subscription that already accepted this key is not run again.
func DeliveryKey(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
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
		IssueID:        eventIssueID(eventType, data),
	}, nil
}

// ResolveEventTeam fills the team on events that do not carry one.
// Attachment payloads identify the parent issue and omit the team. The team
// list resolves a team key. Otherwise SuperPlane reads the issue from Linear.
func ResolveEventTeam(httpCtx core.HTTPContext, integration core.IntegrationContext, event AppEvent, teams []Team) (AppEvent, error) {
	if event.TeamID != "" {
		return event, nil
	}
	if teamID := teamIDForKey(teams, event.TeamKey); teamID != "" {
		event.TeamID = teamID
		return event, nil
	}

	issueID := strings.TrimSpace(event.IssueID)
	if issueID == "" {
		return event, nil
	}

	client, err := NewClient(httpCtx, integration)
	if err != nil {
		return event, err
	}

	issue, err := client.GetIssue(issueID)
	if err != nil {
		return event, err
	}
	if issue.Team == nil {
		return event, nil
	}

	event.TeamID = issue.Team.ID
	if event.TeamKey == "" {
		event.TeamKey = issue.Team.Key
	}
	return event, nil
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
	if key := teamKey(issue); key != "" {
		return key
	}
	if key := teamKeyFromIdentifier(webhookString(issue, "identifier")); key != "" {
		return key
	}
	if key := teamKeyFromIdentifier(identifierFromLinearIssueURL(webhookString(issue, "url"))); key != "" {
		return key
	}
	return ""
}

func eventIssueID(resourceType string, data map[string]any) string {
	if data == nil {
		return ""
	}
	if strings.EqualFold(resourceType, IssueResourceType) {
		return webhookString(data, "id")
	}
	if issueID := webhookString(data, "issueId"); issueID != "" {
		return issueID
	}
	issue, _ := data["issue"].(map[string]any)
	return webhookString(issue, "id")
}

// teamKeyFromIdentifier reads the team key from a Linear issue identifier such as ENG-142.
func teamKeyFromIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	dash := strings.LastIndex(identifier, "-")
	if dash <= 0 || dash == len(identifier)-1 {
		return ""
	}
	for _, char := range identifier[dash+1:] {
		if char < '0' || char > '9' {
			return ""
		}
	}
	return identifier[:dash]
}

func identifierFromLinearIssueURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "issue" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return ""
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
