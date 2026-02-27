package loki

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Loki to work with SuperPlane:

1. **Get Loki URL**: Obtain the base URL of your Loki instance (e.g., ` + "`http://loki:3100`" + `)
2. **Authentication (optional)**: If your Loki instance requires authentication, provide the appropriate credentials:
   - **Basic Auth**: Provide a username and password
   - **Bearer Token**: Provide a bearer token for token-based authentication
   - **Tenant ID**: If using multi-tenancy, provide the X-Scope-OrgID header value
3. **Enter Configuration**: Provide the Loki URL and optional authentication details in the integration configuration
`

func init() {
	registry.RegisterIntegration("loki", &Loki{})
}

type Loki struct{}

type Configuration struct {
	URL      string `json:"url"`
	AuthType string `json:"authType"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
	TenantID string `json:"tenantId"`
}

func (l *Loki) Name() string {
	return "loki"
}

func (l *Loki) Label() string {
	return "Loki"
}

func (l *Loki) Icon() string {
	return "file-text"
}

func (l *Loki) Description() string {
	return "Push and query logs in Grafana Loki"
}

func (l *Loki) Instructions() string {
	return installationInstructions
}

func (l *Loki) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "Loki URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The base URL of your Loki instance",
			Placeholder: "http://loki:3100",
		},
		{
			Name:     "authType",
			Label:    "Authentication Type",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "none",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "none"},
						{Label: "Basic Auth", Value: "basic"},
						{Label: "Bearer Token", Value: "bearer"},
					},
				},
			},
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Username for basic authentication",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"basic"}},
			},
		},
		{
			Name:        "password",
			Label:       "Password",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Password for basic authentication",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"basic"}},
			},
		},
		{
			Name:        "token",
			Label:       "Bearer Token",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Bearer token for token-based authentication",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{"bearer"}},
			},
		},
		{
			Name:        "tenantId",
			Label:       "Tenant ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "X-Scope-OrgID header value for multi-tenant Loki deployments",
			Placeholder: "my-tenant",
		},
	}
}

func (l *Loki) Components() []core.Component {
	return []core.Component{
		&PushLogs{},
		&QueryLogs{},
	}
}

func (l *Loki) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (l *Loki) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *Loki) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.URL == "" {
		return fmt.Errorf("url is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.Ping()
	if err != nil {
		return fmt.Errorf("failed to connect to Loki: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (l *Loki) HandleRequest(ctx core.HTTPRequestContext) {
}

func (l *Loki) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (l *Loki) Actions() []core.Action {
	return []core.Action{}
}

func (l *Loki) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
