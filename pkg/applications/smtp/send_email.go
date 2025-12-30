package smtp

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/wneessen/go-mail"
)

type SendEmail struct{}

type SendEmailConfiguration struct {
	From        string   `json:"from" mapstructure:"from"`
	To          []string `json:"to" mapstructure:"to"`
	Subject     string   `json:"subject" mapstructure:"subject"`
	ContentType string   `json:"contentType" mapstructure:"contentType"`
	Body        string   `json:"body" mapstructure:"body"`
}

func (s *SendEmail) Name() string {
	return "smtp.sendEmail"
}

func (s *SendEmail) Label() string {
	return "Send Email"
}

func (s *SendEmail) Description() string {
	return "Send an email"
}

func (s *SendEmail) Icon() string {
	return "mail"
}

func (s *SendEmail) Color() string {
	return "gray"
}

func (s *SendEmail) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (s *SendEmail) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "from",
			Label:    "From",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "to",
			Label:    "To",
			Type:     configuration.FieldTypeList,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Recipient",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:     "subject",
			Label:    "Subject",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:    "contentType",
			Label:   "Content Type",
			Type:    configuration.FieldTypeSelect,
			Default: "text/plain",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Text",
							Value: "text/plain",
						},
						{
							Label: "HTML",
							Value: "text/html",
						},
					},
				},
			},
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: true,
		},
	}
}

func (s *SendEmail) Setup(ctx core.SetupContext) error {
	// Validate configuration
	var config SendEmailConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.From == "" {
		return fmt.Errorf("from address is required")
	}

	if len(config.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if config.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	return nil
}

func (s *SendEmail) Execute(ctx core.ExecutionContext) error {
	var config SendEmailConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Create SMTP client
	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	// Create the email message
	msg := mail.NewMsg()
	if err := msg.From(config.From); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}

	if err := msg.To(config.To...); err != nil {
		return fmt.Errorf("failed to set to addresses: %w", err)
	}

	msg.Subject(config.Subject)
	msg.SetBodyString(mail.ContentType(config.ContentType), config.Body)

	// Send email with timeout
	sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.DialAndSendWithContext(sendCtx, msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		"smtp.email",
		[]any{map[string]any{
			"from":    config.From,
			"to":      config.To,
			"subject": config.Subject,
		}},
	)
}

func (s *SendEmail) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (s *SendEmail) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (s *SendEmail) Actions() []core.Action {
	return []core.Action{}
}

func (s *SendEmail) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (s *SendEmail) Cancel(ctx core.ExecutionContext) error {
	return nil
}
