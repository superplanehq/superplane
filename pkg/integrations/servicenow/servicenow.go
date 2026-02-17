package servicenow

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
	registry.RegisterIntegration("servicenow", &ServiceNow{})
}

const (
	AuthTypeBasicAuth = "basicAuth"
	AuthTypeOAuth     = "oauth"
	OAuthAccessToken  = "accessToken"
)

type ServiceNow struct{}

type Configuration struct {
	InstanceURL  string  `json:"instanceUrl"`
	AuthType     string  `json:"authType"`
	Username     *string `json:"username"`
	Password     *string `json:"password"`
	ClientID     *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
}

func (s *ServiceNow) Name() string {
	return "servicenow"
}

func (s *ServiceNow) Label() string {
	return "ServiceNow"
}

func (s *ServiceNow) Icon() string {
	return "servicenow"
}

func (s *ServiceNow) Description() string {
	return "Manage and react to incidents in ServiceNow"
}

func (s *ServiceNow) Instructions() string {
	return `Requires a ServiceNow instance with API access. The following roles are needed on your ServiceNow instance:

**Integration account** (for Basic Auth or OAuth):
- **itil** — read/write access to the Incident table

Optionally, enable **Web Service Access Only** on the integration account to restrict it to API-only use.

**On Incident trigger**: Setting up the Business Rule on ServiceNow requires the **admin** role. This is a one-time setup.`
}

func (s *ServiceNow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceUrl",
			Label:       "Instance URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your ServiceNow instance URL (e.g. https://dev12345.service-now.com)",
			Placeholder: "https://dev12345.service-now.com",
		},
		{
			Name:     "authType",
			Label:    "Auth Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AuthTypeBasicAuth,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Basic Auth", Value: AuthTypeBasicAuth},
						{Label: "OAuth", Value: AuthTypeOAuth},
					},
				},
			},
		},
		{
			Name:     "username",
			Label:    "Username",
			Type:     configuration.FieldTypeString,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeBasicAuth}},
			},
		},
		{
			Name:      "password",
			Label:     "Password",
			Type:      configuration.FieldTypeString,
			Required:  true,
			Sensitive: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeBasicAuth}},
			},
		},
		{
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OAuth Client ID from your ServiceNow instance",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeOAuth}},
			},
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "OAuth Client Secret from your ServiceNow instance",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeOAuth}},
			},
		},
	}
}

func (s *ServiceNow) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (s *ServiceNow) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
	}
}

func (s *ServiceNow) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *ServiceNow) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.InstanceURL == "" {
		return fmt.Errorf("instanceUrl is required")
	}

	if config.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	if config.AuthType != AuthTypeBasicAuth && config.AuthType != AuthTypeOAuth {
		return fmt.Errorf("authType %s is not supported", config.AuthType)
	}

	if config.AuthType == AuthTypeOAuth {
		return s.oauthSync(ctx, config)
	}

	return s.basicAuthSync(ctx, config)
}

func (s *ServiceNow) basicAuthSync(ctx core.SyncContext, config Configuration) error {
	if config.Username == nil || *config.Username == "" {
		return fmt.Errorf("username is required")
	}

	if config.Password == nil || *config.Password == "" {
		return fmt.Errorf("password is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateConnection()
	if err != nil {
		return fmt.Errorf("error validating credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (s *ServiceNow) oauthSync(ctx core.SyncContext, config Configuration) error {
	if config.ClientID == nil || *config.ClientID == "" {
		return fmt.Errorf("clientId is required")
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	tokenURL := fmt.Sprintf("%s/oauth_token.do", strings.TrimRight(config.InstanceURL, "/"))
	r, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(*config.ClientID, string(clientSecret))
	resp, err := ctx.HTTP.Do(r)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error generating access token: request got %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %v", err)
	}

	err = ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken))
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateConnection()
	if err != nil {
		return fmt.Errorf("error validating credentials: %v", err)
	}

	ctx.Integration.Ready()

	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (r *TokenResponse) GetExpiration() time.Duration {
	if r.ExpiresIn > 0 {
		return time.Duration(r.ExpiresIn/2) * time.Second
	}

	return time.Hour
}

func (s *ServiceNow) HandleRequest(ctx core.HTTPRequestContext) {}

func (s *ServiceNow) Actions() []core.Action {
	return []core.Action{}
}

func (s *ServiceNow) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
