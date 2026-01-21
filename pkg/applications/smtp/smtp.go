package smtp

import (
	"fmt"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("smtp", &SMTP{})
}

type SMTP struct{}

type Configuration struct {
	Host      string `json:"host" mapstructure:"host"`
	Port      string `json:"port" mapstructure:"port"`
	Username  string `json:"username" mapstructure:"username"`
	Password  string `json:"password" mapstructure:"password"`
	FromName  string `json:"fromName" mapstructure:"fromName"`
	FromEmail string `json:"fromEmail" mapstructure:"fromEmail"`
	UseTLS    bool   `json:"useTLS" mapstructure:"useTLS"`
}

func (s *SMTP) Name() string {
	return "smtp"
}

func (s *SMTP) Label() string {
	return "SMTP"
}

func (s *SMTP) Icon() string {
	return "smtp"
}

func (s *SMTP) Description() string {
	return "Send emails via any SMTP server"
}

func (s *SMTP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "host",
			Label:       "SMTP Host",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "SMTP server hostname (e.g., smtp.gmail.com, smtp.sendgrid.net)",
		},
		{
			Name:        "port",
			Label:       "SMTP Port",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "587",
			Description: "SMTP server port (commonly 587 for TLS, 465 for SSL, or 25)",
		},
		{
			Name:        "username",
			Label:       "Username",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "SMTP authentication username (often your email address or API key name)",
		},
		{
			Name:        "password",
			Label:       "Password",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "SMTP authentication password or API key",
		},
		{
			Name:        "fromName",
			Label:       "From Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Default sender display name",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Default sender email address",
		},
		{
			Name:        "useTLS",
			Label:       "Use TLS",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Enable STARTTLS encryption (recommended)",
		},
	}
}

func (s *SMTP) Components() []core.Component {
	return []core.Component{
		&SendEmail{},
	}
}

func (s *SMTP) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (s *SMTP) InstallationInstructions() string {
	return ""
}

func (s *SMTP) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.Host == "" {
		return fmt.Errorf("host is required")
	}

	port, err := strconv.Atoi(config.Port)
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}

	if config.FromEmail == "" {
		return fmt.Errorf("fromEmail is required")
	}

	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("SMTP connection test failed: %w", err)
	}

	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (s *SMTP) HandleRequest(ctx core.HTTPRequestContext) {
	// SMTP doesn't handle incoming webhooks
}

func (s *SMTP) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (s *SMTP) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	// SMTP doesn't have resources to list
	return []core.ApplicationResource{}, nil
}

func (s *SMTP) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (s *SMTP) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
