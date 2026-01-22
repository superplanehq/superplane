package pagerduty

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

/*
 * 1. Integrations > App Registration > New App
 * 2. Set the name and description for new app
 * 3. Functionality -> select "OAuth 2.0" only
 * 4. Authorization -> "Scoped Auth"
 * 5. Permission Scope ->
 *   - incidents.read, incidents.write
 *   - webhook_subscriptions.read, webhook_subscriptions.write
 *   - users.read
 *   - teams.read
 *   - services.read
 *   - schedules.read
 *   - priorities.read
 *   - oncalls.read
 *   - incident_types.read
 *   - escalation_policies.read
 *   - custom_fields.read
 */

func init() {
	registry.RegisterApplication("pagerduty", &PagerDuty{})
}

type PagerDuty struct{}

const (
	AuthTypeAPIToken = "apiToken"
	AuthTypeAppOAuth = "appOAuth"
	AppAccessToken   = "accessToken"
)

type Configuration struct {
	AuthType     string  `json:"authType"`
	Region       string  `json:"region"`
	SubDomain    string  `json:"subdomain"`
	APIToken     *string `json:"apiToken"`
	ClientID     *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
}

type Metadata struct {
	Services []Service `json:"services"`
}

func (p *PagerDuty) Name() string {
	return "pagerduty"
}

func (p *PagerDuty) Label() string {
	return "PagerDuty"
}

func (p *PagerDuty) Icon() string {
	return "alert-triangle"
}

func (p *PagerDuty) Description() string {
	return "Manage and react to incidents in PagerDuty"
}

func (p *PagerDuty) InstallationInstructions() string {
	return ""
}

func (p *PagerDuty) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US", Value: "us"},
						{Label: "EU", Value: "eu"},
					},
				},
			},
		},
		{
			Name:     "subdomain",
			Label:    "Sub Domain",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "authType",
			Label:    "Auth Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AuthTypeAPIToken,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "API Token", Value: AuthTypeAPIToken},
						{Label: "App OAuth", Value: AuthTypeAppOAuth},
					},
				},
			},
		},
		{
			Name:      "apiToken",
			Label:     "API Token",
			Type:      configuration.FieldTypeString,
			Required:  true,
			Sensitive: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAPIToken}},
			},
		},
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OAuth Client ID from your PagerDuty App",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAppOAuth}},
			},
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "OAuth Client Secret from your PagerDuty App",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAppOAuth}},
			},
		},
	}
}

func (p *PagerDuty) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
		&UpdateIncident{},
	}
}

func (p *PagerDuty) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
		&OnIncidentStatusUpdate{},
	}
}

func (p *PagerDuty) Sync(ctx core.SyncContext) error {
	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if configuration.Region == "" {
		return fmt.Errorf("region is required")
	}

	if configuration.SubDomain == "" {
		return fmt.Errorf("subdomain is required")
	}

	if configuration.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	if configuration.AuthType != AuthTypeAPIToken && configuration.AuthType != AuthTypeAppOAuth {
		return fmt.Errorf("authType %s is not supported", configuration.AuthType)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If App OAuth is used, we need to generate the access token.
	//
	if configuration.AuthType == AuthTypeAppOAuth {
		return p.appOAuthSync(ctx, configuration)
	}

	return p.apiTokenSync(ctx)
}

func (p *PagerDuty) apiTokenSync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	services, err := client.ListServices()
	if err != nil {
		return fmt.Errorf("error listing services: %v", err)
	}

	ctx.AppInstallation.SetMetadata(Metadata{Services: services})
	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (p *PagerDuty) appOAuthSync(ctx core.SyncContext, configuration Configuration) error {
	if configuration.ClientID == nil || *configuration.ClientID == "" {
		return fmt.Errorf("clientId is required")
	}

	clientSecret, err := ctx.AppInstallation.GetConfig("clientSecret")
	if err != nil {
		return err
	}

	scopes := []string{
		fmt.Sprintf("as_account-%s.%s", configuration.Region, configuration.SubDomain),
		"custom_fields.read",
		"escalation_policies.read",
		"incident_types.read",
		"incidents.read",
		"incidents.write",
		"oncalls.read",
		"priorities.read",
		"schedules.read",
		"services.read",
		"teams.read",
		"users.read",
		"webhook_subscriptions.read",
		"webhook_subscriptions.write",
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", *configuration.ClientID)
	data.Set("client_secret", string(clientSecret))
	data.Set("scope", strings.Join(scopes, " "))

	r, err := http.NewRequest(http.MethodPost, "https://identity.pagerduty.com/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := ctx.HTTP.Do(r)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error generating access token for app: request got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %v", err)
	}

	err = ctx.AppInstallation.SetSecret(AppAccessToken, []byte(tokenResponse.AccessToken))
	if err != nil {
		return err
	}

	//
	// Verify that the auth is working by listing the services.
	//
	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client")
	}

	services, err := client.ListServices()
	if err != nil {
		return fmt.Errorf("error determing abilities: %v", err)
	}

	ctx.AppInstallation.SetMetadata(Metadata{Services: services})
	ctx.AppInstallation.SetState("ready", "")

	//
	// Schedule a new sync to refresh the access token before it expires
	//
	return ctx.AppInstallation.ScheduleResync(tokenResponse.GetExpiration())
}

func (p *PagerDuty) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

func (r *TokenResponse) GetExpiration() time.Duration {
	if r.ExpiresIn > 0 {
		return time.Duration(r.ExpiresIn/2) * time.Second
	}

	return time.Hour
}

type WebhookConfiguration struct {

	//
	// Specific event types, e.g., ["incident.resolved", "incident.triggered"]
	//
	Events []string `json:"events"`

	//
	// Filter for webhook.
	//
	Filter WebhookFilter `json:"filter"`
}

type WebhookFilter struct {
	//
	// Type of filter for event subscription:
	// - account_reference: webhook is created on account level
	// - team_reference: events will be sent only for events related to the specified team
	// - service_reference: events will be sent only for events related ot the specified service.
	//
	Type string `json:"type"`

	//
	// If team_reference is used, this must be the ID of a team.
	// If service_reference is used, this must be the ID of a service.
	//
	ID string `json:"id"`
}

type WebhookMetadata struct {
	SubscriptionID string `json:"subscriptionId"`
}

func (p *PagerDuty) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	//
	// The event subscription filter on the webhook must match exactly
	//
	if configA.Filter.Type != configB.Filter.Type || configA.Filter.ID != configB.Filter.ID {
		return false, nil
	}

	// Check if A contains all events from B (A is superset of B)
	// This allows webhook sharing when existing webhook has more events than needed
	for _, eventB := range configB.Events {
		if !slices.Contains(configA.Events, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (p *PagerDuty) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	//
	// Create webhook subscription.
	// NOTE: PagerDuty returns the secret used for signing webhooks
	// on the subscription response, so we need to update the webhook secret on our end.
	//
	subscription, err := client.CreateWebhookSubscription(ctx.Webhook.GetURL(), configuration.Events, configuration.Filter)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook subscription: %v", err)
	}

	err = ctx.Webhook.SetSecret([]byte(subscription.DeliveryMethod.Secret))
	if err != nil {
		return nil, fmt.Errorf("error updating webhook secret: %v", err)
	}

	return WebhookMetadata{
		SubscriptionID: subscription.ID,
	}, nil
}

func (p *PagerDuty) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return err
	}

	err = client.DeleteWebhookSubscription(metadata.SubscriptionID)
	if err != nil {
		return fmt.Errorf("error deleting webhook subscription: %v", err)
	}

	return nil
}
