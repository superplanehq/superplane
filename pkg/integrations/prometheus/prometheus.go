package prometheus

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

	AlertStateAny      = "any"
	AlertStateFiring   = "firing"
	AlertStateResolved = "resolved"
	AlertStatePending  = "pending"
	AlertStateInactive = "inactive"

	PrometheusAlertPayloadType = "prometheus.alert"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("prometheus", &Prometheus{}, &PrometheusWebhookHandler{})
}

type Prometheus struct{}

type Configuration struct {
	BaseURL            string `json:"baseURL" mapstructure:"baseURL"`
	AlertmanagerURL    string `json:"alertmanagerURL,omitempty" mapstructure:"alertmanagerURL"`
	AuthType           string `json:"authType" mapstructure:"authType"`
	Username           string `json:"username,omitempty" mapstructure:"username"`
	Password           string `json:"password,omitempty" mapstructure:"password"`
	BearerToken        string `json:"bearerToken,omitempty" mapstructure:"bearerToken"`
	WebhookBearerToken string `json:"webhookBearerToken,omitempty" mapstructure:"webhookBearerToken"`
}

type Metadata struct{}

func (p *Prometheus) Name() string {
	return "prometheus"
}

func (p *Prometheus) Label() string {
	return "Prometheus"
}

func (p *Prometheus) Icon() string {
	return "prometheus"
}

func (p *Prometheus) Description() string {
	return "Monitor alerts from Prometheus and Alertmanager"
}

func (p *Prometheus) Instructions() string {
	return `### Connection

Configure this integration with:
- **Prometheus Base URL**: URL of your Prometheus server (e.g., ` + "`https://prometheus.example.com`" + `)
- **Alertmanager Base URL** (optional): URL of your Alertmanager instance (e.g., ` + "`https://alertmanager.example.com`" + `). Required for Silence components. If omitted, the Prometheus Base URL is used.
- **API Auth**: ` + "`none`" + `, ` + "`basic`" + `, or ` + "`bearer`" + ` for API requests
- **Webhook Secret** (recommended): If set, Alertmanager must send ` + "`Authorization: Bearer <token>`" + ` on webhook requests

### Alertmanager Setup (manual)

The trigger setup panel in SuperPlane shows the generated webhook URL.
Use the On Alert trigger setup instructions in the workflow sidebar for the exact ` + "`alertmanager.yml`" + ` snippet.

After editing config, reload Alertmanager (for example ` + "`POST /-/reload`" + ` when lifecycle reload is enabled).`
}

func (p *Prometheus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "Prometheus Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "https://prometheus.example.com",
			Description: "Base URL for Prometheus HTTP API",
		},
		{
			Name:        "alertmanagerURL",
			Label:       "Alertmanager Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "https://alertmanager.example.com",
			Description: "Base URL for Alertmanager API (used by Silence components). Falls back to Prometheus Base URL if not set.",
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
		{
			Name:        "webhookBearerToken",
			Label:       "Webhook Secret",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Secret required by incoming Alertmanager webhooks. Recommended for production environments.",
		},
	}
}

func (p *Prometheus) Components() []core.Component {
	return []core.Component{
		&GetAlert{},
		&CreateSilence{},
		&ExpireSilence{},
	}
}

func (p *Prometheus) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlert{},
	}
}

func (p *Prometheus) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateIntegrationConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	if _, err := client.Query("up"); err != nil {
		if _, fallbackErr := client.GetAlertsFromPrometheus(); fallbackErr != nil {
			return fmt.Errorf("error validating connection: query failed (%v), alerts failed (%v)", err, fallbackErr)
		}
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

func validateIntegrationConfiguration(config Configuration) error {
	if config.BaseURL == "" {
		return fmt.Errorf("baseURL is required")
	}

	authType := config.AuthType
	switch authType {
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

func (p *Prometheus) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (p *Prometheus) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (p *Prometheus) Actions() []core.Action {
	return []core.Action{}
}

func (p *Prometheus) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
