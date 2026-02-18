package opencost

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

	CostAllocationPayloadType = "opencost.costAllocation"
)

func init() {
	registry.RegisterIntegration("opencost", &OpenCost{})
}

type OpenCost struct{}

type Configuration struct {
	BaseURL     string `json:"baseURL" mapstructure:"baseURL"`
	AuthType    string `json:"authType" mapstructure:"authType"`
	Username    string `json:"username,omitempty" mapstructure:"username"`
	Password    string `json:"password,omitempty" mapstructure:"password"`
	BearerToken string `json:"bearerToken,omitempty" mapstructure:"bearerToken"`
}

type Metadata struct{}

func (o *OpenCost) Name() string {
	return "opencost"
}

func (o *OpenCost) Label() string {
	return "OpenCost"
}

func (o *OpenCost) Icon() string {
	return "opencost"
}

func (o *OpenCost) Description() string {
	return "Monitor Kubernetes cost allocation with OpenCost"
}

func (o *OpenCost) Instructions() string {
	return `### Connection

Configure this integration with:
- **OpenCost API URL**: URL of your OpenCost API server (e.g., ` + "`http://opencost.example.com:9003`" + `)
- **API Auth**: ` + "`none`" + `, ` + "`basic`" + `, or ` + "`bearer`" + ` depending on how your OpenCost instance is secured

### Finding the OpenCost API URL

If OpenCost is running in Kubernetes, you can port-forward:
` + "```" + `
kubectl port-forward -n opencost svc/opencost 9003:9003
` + "```" + `

Then use ` + "`http://localhost:9003`" + ` as the base URL.

For production, use the externally accessible URL of your OpenCost API.`
}

func (o *OpenCost) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "OpenCost API URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "http://opencost.example.com:9003",
			Description: "Base URL for the OpenCost API",
		},
		{
			Name:     "authType",
			Label:    "API Auth Type",
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
	}
}

func (o *OpenCost) Components() []core.Component {
	return []core.Component{
		&GetCostAllocation{},
	}
}

func (o *OpenCost) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnCostExceedsThreshold{},
	}
}

func (o *OpenCost) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateIntegrationConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OpenCost client: %w", err)
	}

	if _, err := client.GetAllocation("1h", "namespace"); err != nil {
		return fmt.Errorf("error validating connection: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

func validateIntegrationConfiguration(config Configuration) error {
	if config.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}

	switch config.AuthType {
	case AuthTypeNone:
	case AuthTypeBasic:
		if config.Username == "" {
			return fmt.Errorf("username is required when authType is basic")
		}
		if config.Password == "" {
			return fmt.Errorf("password is required when authType is basic")
		}
	case AuthTypeBearer:
		if config.BearerToken == "" {
			return fmt.Errorf("bearerToken is required when authType is bearer")
		}
	default:
		return fmt.Errorf("authType must be one of: none, basic, bearer")
	}

	return nil
}

func (o *OpenCost) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenCost) HandleRequest(ctx core.HTTPRequestContext) {
}

func (o *OpenCost) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (o *OpenCost) Actions() []core.Action {
	return []core.Action{}
}

func (o *OpenCost) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
