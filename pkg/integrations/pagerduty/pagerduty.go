package pagerduty

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("pagerduty", &PagerDuty{}, &PagerDutyWebhookHandler{})
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

func (p *PagerDuty) Instructions() string {
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
		&AcknowledgeIncident{},
		&ResolveIncident{},
		&EscalateIncident{},
		&AnnotateIncident{},
		&ListIncidents{},
		&ListNotes{},
		&ListLogEntries{},
		&SnoozeIncident{},
	}
}

func (p *PagerDuty) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
		&OnIncidentStatusUpdate{},
		&OnIncidentAnnotated{},
	}
}

func (p *PagerDuty) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
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
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
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
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	services, err := client.ListServices()
	if err != nil {
		return fmt.Errorf("error listing services: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Services: services})
	ctx.Integration.Ready()
	return nil
}

func (p *PagerDuty) appOAuthSync(ctx core.SyncContext, configuration Configuration) error {
	if configuration.ClientID == nil || *configuration.ClientID == "" {
		return fmt.Errorf("clientId is required")
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
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

	err = ctx.Integration.SetSecret(AppAccessToken, []byte(tokenResponse.AccessToken))
	if err != nil {
		return err
	}

	//
	// Verify that the auth is working by listing the services.
	//
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client")
	}

	services, err := client.ListServices()
	if err != nil {
		return fmt.Errorf("error determing abilities: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Services: services})
	ctx.Integration.Ready()

	//
	// Schedule a new sync to refresh the access token before it expires
	//
	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
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

func (p *PagerDuty) Actions() []core.Action {
	return []core.Action{}
}

func (p *PagerDuty) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
