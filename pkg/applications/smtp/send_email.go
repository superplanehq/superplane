package smtp

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendEmail struct{}

type SendEmailConfiguration struct {
	To        string `json:"to" mapstructure:"to"`
	Cc        string `json:"cc" mapstructure:"cc"`
	Bcc       string `json:"bcc" mapstructure:"bcc"`
	Subject   string `json:"subject" mapstructure:"subject"`
	Body      string `json:"body" mapstructure:"body"`
	IsHTML    bool   `json:"isHTML" mapstructure:"isHTML"`
	FromName  string `json:"fromName" mapstructure:"fromName"`
	FromEmail string `json:"fromEmail" mapstructure:"fromEmail"`
	ReplyTo   string `json:"replyTo" mapstructure:"replyTo"`
}

type SendEmailMetadata struct {
	To      []string `json:"to" mapstructure:"to"`
	Subject string   `json:"subject" mapstructure:"subject"`
}

func (c *SendEmail) Name() string {
	return "smtp.sendEmail"
}

func (c *SendEmail) Label() string {
	return "Send Email"
}

func (c *SendEmail) Description() string {
	return "Send an email via SMTP"
}

func (c *SendEmail) Icon() string {
	return "smtp"
}

func (c *SendEmail) Color() string {
	return "gray"
}

func (c *SendEmail) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendEmail) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Recipient email addresses (comma-separated for multiple)",
		},
		{
			Name:        "cc",
			Label:       "CC",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "CC recipients (comma-separated)",
		},
		{
			Name:        "bcc",
			Label:       "BCC",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "BCC recipients (comma-separated)",
		},
		{
			Name:        "subject",
			Label:       "Subject",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Email subject line",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Email body content",
		},
		{
			Name:        "isHTML",
			Label:       "HTML Format",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enable if the body contains HTML markup",
		},
		{
			Name:        "fromName",
			Label:       "From Name (Override)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the default sender display name",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email (Override)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Override the default sender email address",
		},
		{
			Name:        "replyTo",
			Label:       "Reply-To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Reply-to email address",
		},
	}
}

func (c *SendEmail) Setup(ctx core.SetupContext) error {
	var config SendEmailConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.To == "" {
		return fmt.Errorf("to is required")
	}

	if config.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	// Validate email addresses
	toAddrs, err := parseEmailList(config.To)
	if err != nil {
		return fmt.Errorf("invalid 'to' email addresses: %w", err)
	}

	if config.Cc != "" {
		if _, err := parseEmailList(config.Cc); err != nil {
			return fmt.Errorf("invalid 'cc' email addresses: %w", err)
		}
	}

	if config.Bcc != "" {
		if _, err := parseEmailList(config.Bcc); err != nil {
			return fmt.Errorf("invalid 'bcc' email addresses: %w", err)
		}
	}

	if config.FromEmail != "" {
		if _, err := mail.ParseAddress(config.FromEmail); err != nil {
			return fmt.Errorf("invalid 'fromEmail' address: %w", err)
		}
	}

	if config.ReplyTo != "" {
		if _, err := mail.ParseAddress(config.ReplyTo); err != nil {
			return fmt.Errorf("invalid 'replyTo' address: %w", err)
		}
	}

	metadata := SendEmailMetadata{
		To:      toAddrs,
		Subject: config.Subject,
	}

	return ctx.Metadata.Set(metadata)
}

func (c *SendEmail) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendEmail) Execute(ctx core.ExecutionContext) error {
	var config SendEmailConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.To == "" {
		return fmt.Errorf("to is required")
	}

	if config.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	toAddrs, _ := parseEmailList(config.To)
	ccAddrs, _ := parseEmailList(config.Cc)
	bccAddrs, _ := parseEmailList(config.Bcc)

	// Set text/html body based on isHTML flag
	var textBody, htmlBody string
	if config.IsHTML {
		htmlBody = config.Body
	} else {
		textBody = config.Body
	}

	email := Email{
		To:        toAddrs,
		Cc:        ccAddrs,
		Bcc:       bccAddrs,
		Subject:   config.Subject,
		TextBody:  textBody,
		HTMLBody:  htmlBody,
		FromName:  config.FromName,
		FromEmail: config.FromEmail,
		ReplyTo:   config.ReplyTo,
	}

	result, err := client.SendEmail(email)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"smtp.email.sent",
		[]any{result},
	)
}

func (c *SendEmail) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *SendEmail) Actions() []core.Action {
	return []core.Action{}
}

func (c *SendEmail) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *SendEmail) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// parseEmailList parses a comma-separated list of email addresses
func parseEmailList(emails string) ([]string, error) {
	if emails == "" {
		return []string{}, nil
	}

	parts := strings.Split(emails, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			continue
		}

		// Validate email format
		if _, err := mail.ParseAddress(addr); err != nil {
			return nil, fmt.Errorf("invalid email address '%s': %w", addr, err)
		}

		result = append(result, addr)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid email addresses found")
	}

	return result, nil
}
