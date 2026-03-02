package loki

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	AuthTypeNone   = "none"
	AuthTypeBasic  = "basic"
	AuthTypeBearer = "bearer"
)

const installationInstructions = `### Connection

1. **Loki URL**: Provide the base URL of your Loki instance (e.g. ` + "`https://loki.example.com`" + `)
2. **Authentication**: Choose the authentication method:
   - **None**: No authentication (e.g. local or internal Loki)
   - **Basic**: Username and password (e.g. Grafana Cloud)
   - **Bearer**: Bearer token (e.g. Loki behind an auth proxy)
3. **Tenant ID** (optional): If your Loki instance is multi-tenant, provide the tenant ID (sent as ` + "`X-Scope-OrgID`" + ` header)
`

func init() {
	registry.RegisterIntegration("loki", &Loki{})
}

type Loki struct{}

type Configuration struct {
	BaseURL     string `json:"baseURL" mapstructure:"baseURL"`
	AuthType    string `json:"authType" mapstructure:"authType"`
	Username    string `json:"username,omitempty" mapstructure:"username"`
	Password    string `json:"password,omitempty" mapstructure:"password"`
	BearerToken string `json:"bearerToken,omitempty" mapstructure:"bearerToken"`
	TenantID    string `json:"tenantID,omitempty" mapstructure:"tenantID"`
}

func (l *Loki) Name() string {
	return "loki"
}

func (l *Loki) Label() string {
	return "Loki"
}

func (l *Loki) Icon() string {
	return "loki"
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
			Name:        "baseURL",
			Label:       "Loki Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://loki.example.com",
			Description: "Base URL of your Loki instance",
		},
		{
			Name:     "authType",
			Label:    "Authentication",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  AuthTypeNone,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: AuthTypeNone},
						{Label: "Basic", Value: AuthTypeBasic},
						{Label: "Bearer", Value: AuthTypeBearer},
					},
				},
			},
		},
		{
			Name:     "username",
			Label:    "Username",
			Type:     configuration.FieldTypeString,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeBasic}},
			},
		},
		{
			Name:      "password",
			Label:     "Password",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Sensitive: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeBasic}},
			},
		},
		{
			Name:      "bearerToken",
			Label:     "Bearer Token",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Sensitive: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeBearer}},
			},
		},
		{
			Name:        "tenantID",
			Label:       "Tenant ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "X-Scope-OrgID header value for multi-tenant Loki deployments",
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
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}

	if config.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.Ready(); err != nil {
		return fmt.Errorf("failed to verify connection: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (l *Loki) HandleRequest(ctx core.HTTPRequestContext) {}

func (l *Loki) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (l *Loki) Actions() []core.Action {
	return []core.Action{}
}

func (l *Loki) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
