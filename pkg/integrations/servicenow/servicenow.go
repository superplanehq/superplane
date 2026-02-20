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
	OAuthAccessToken = "accessToken"
)

type ServiceNow struct{}

type Configuration struct {
	InstanceURL  string  `json:"instanceUrl"`
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
	return `Requires a ServiceNow instance with OAuth API access.

Before creating OAuth credentials, enable client credentials grant on your instance:
- Go to **System Properties** (sys_properties_list.do) and search for:
  - **Name**: glide.oauth.inbound.client.credential.grant_type.enabled
  - (Important: the property name ends with **enabled**)
- If it does not exist, create it with:
  - **Application Scope**: Global
  - **Type**: true | false
  - **Value**: true

Then configure OAuth:
- Go to **System OAuth > Inbound Integrations**
- Create a new integration with **OAuth - Client Credentials Grant**
- Copy the generated **Client ID** and **Client Secret**
- Assign required permissions to the integration account:
  - **itil** role (required for incident read/write)
  - Optionally **admin** if broader scoped access is needed
- Optionally enable **Web Service Access Only** on the integration account to restrict it to API-only use.`
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
			Name:        "clientId",
			Label:       "Client ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OAuth Client ID from your ServiceNow instance",
		},
		{
			Name:        "clientSecret",
			Label:       "Client Secret",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "OAuth Client Secret from your ServiceNow instance",
		},
	}
}

func (s *ServiceNow) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
		&GetIncident{},
	}
}

func (s *ServiceNow) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (s *ServiceNow) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *ServiceNow) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if config.InstanceURL == "" {
		return fmt.Errorf("instanceUrl is required")
	}

	if config.ClientID == nil || *config.ClientID == "" {
		return fmt.Errorf("clientId is required")
	}

	clientSecret, err := ctx.Integration.GetConfig("clientSecret")
	if err != nil {
		return fmt.Errorf("failed to get clientSecret: %w", err)
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	tokenURL := fmt.Sprintf("%s/oauth_token.do", strings.TrimRight(config.InstanceURL, "/"))
	r, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(*config.ClientID, string(clientSecret))
	resp, err := ctx.HTTP.Do(r)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error generating access token: request got %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	err = ctx.Integration.SetSecret(OAuthAccessToken, []byte(tokenResponse.AccessToken))
	if err != nil {
		return fmt.Errorf("failed to set access token secret: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	err = client.ValidateConnection()
	if err != nil {
		return fmt.Errorf("error validating credentials: %w", err)
	}

	metadata, err := fetchMetadata(client)
	if err != nil {
		return fmt.Errorf("failed to fetch integration metadata: %w", err)
	}

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.Ready()

	return ctx.Integration.ScheduleResync(tokenResponse.GetExpiration())
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Metadata struct {
	Categories       []ChoiceRecord          `json:"categories"`
	AssignmentGroups []AssignmentGroupRecord `json:"assignmentGroups"`
}

func fetchMetadata(client *Client) (Metadata, error) {
	categories, err := client.ListCategories()
	if err != nil {
		return Metadata{}, fmt.Errorf("error listing categories: %w", err)
	}

	groups, err := client.ListAssignmentGroups()
	if err != nil {
		return Metadata{}, fmt.Errorf("error listing assignment groups: %w", err)
	}

	return Metadata{
		Categories:       categories,
		AssignmentGroups: groups,
	}, nil
}

func (r *TokenResponse) GetExpiration() time.Duration {
	if r.ExpiresIn > 0 {
		return time.Duration(max(r.ExpiresIn/2, 1)) * time.Second
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
