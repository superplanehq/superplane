package sendgrid

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	SendEmailPayloadType       = "sendgrid.email.sent"
	SendEmailFailedPayloadType = "sendgrid.email.failed"
	SendEmailFailedChannel     = "failed"
)

type SendEmail struct{}

type SendEmailConfiguration struct {
	To           string         `json:"to" mapstructure:"to"`
	Subject      string         `json:"subject" mapstructure:"subject"`
	Body         string         `json:"body" mapstructure:"body"`
	Mode         string         `json:"mode" mapstructure:"mode"`
	HTMLBody     string         `json:"htmlBody" mapstructure:"htmlBody"`
	Cc           string         `json:"cc" mapstructure:"cc"`
	Bcc          string         `json:"bcc" mapstructure:"bcc"`
	FromName     string         `json:"fromName" mapstructure:"fromName"`
	FromEmail    string         `json:"fromEmail" mapstructure:"fromEmail"`
	ReplyTo      string         `json:"replyTo" mapstructure:"replyTo"`
	Categories   string         `json:"categories" mapstructure:"categories"`
	TemplateID   string         `json:"templateId" mapstructure:"templateId"`
	TemplateData map[string]any `json:"templateData" mapstructure:"templateData"`
}

type SendEmailMetadata struct {
	To        []string `json:"to" mapstructure:"to"`
	Subject   string   `json:"subject" mapstructure:"subject"`
	FromEmail string   `json:"fromEmail" mapstructure:"fromEmail"`
}

type SendEmailFailure struct {
	Error        string `json:"error"`
	StatusCode   int    `json:"statusCode,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
}

func (c *SendEmail) Name() string {
	return "sendgrid.sendEmail"
}

func (c *SendEmail) Label() string {
	return "Send Email"
}

func (c *SendEmail) Description() string {
	return "Send an email via SendGrid"
}

func (c *SendEmail) Documentation() string {
	return `Send a single email via SendGrid's Mail Send API.

## Use Cases

- **Notifications**: Send alert or notification emails when workflows fail or complete
- **Receipts**: Send order confirmations or receipts from workflow runs
- **Reports**: Deliver scheduled digests or reports to stakeholders

## Configuration

- **To**: Recipient email address(es), comma-separated
- **Subject**: Email subject line
- **Sending Mode**: Choose text, HTML, or dynamic template
- **Text Body**: Email body content (plain text)
- **HTML Body**: Email body content (HTML)
- **CC**: CC recipients, comma-separated
- **BCC**: BCC recipients, comma-separated
- **From Name**: Optional sender display name override
- **From Email**: Optional sender email override (must be verified in SendGrid)
- **Reply-To**: Reply-to email address
- **Template ID**: SendGrid dynamic template ID (e.g. ` + "`d-xxxxxxxx`" + `)
- **Template Data**: JSON object of template substitution variables

## Output Channels

- **Default**: Emitted when SendGrid accepts the message
- **Failed**: Emitted when validation fails or the API request is rejected

## Notes

- Requires a SendGrid API key configured on the integration
- When using a template, SendGrid may override the subject and body`
}

func (c *SendEmail) Icon() string {
	return "mail"
}

func (c *SendEmail) Color() string {
	return "gray"
}

func (c *SendEmail) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  core.DefaultOutputChannel.Name,
			Label: "Success",
		},
		{
			Name:  SendEmailFailedChannel,
			Label: "Failed",
		},
	}
}

func (c *SendEmail) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Recipient email addresses (comma-separated for multiple)",
		}, {
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
		{
			Name:        "subject",
			Label:       "Subject",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email subject line",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"text", "html"}},
			},
		},
		{
			Name:     "mode",
			Label:    "Sending Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "text",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Text Body", Value: "text"},
						{Label: "HTML Body", Value: "html"},
						{Label: "Template", Value: "template"},
					},
				},
			},
			Description: "Choose how the email content is sent",
		},
		{
			Name:        "body",
			Label:       "Text Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Plain text email body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"text"}},
			},
		},
		{
			Name:        "htmlBody",
			Label:       "HTML Body",
			Type:        configuration.FieldTypeXML,
			Required:    false,
			Description: "HTML email body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"html"}},
			},
		},
		{
			Name:        "templateId",
			Label:       "Template ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "SendGrid dynamic template ID (e.g. d-xxxxxxxx)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"template"}},
			},
		},
		{
			Name:        "templateData",
			Label:       "Template Data",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "JSON object with template variables",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{"template"}},
			},
		},
		{
			Name:        "categories",
			Label:       "Categories",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "SendGrid categories (comma-separated)",
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

	if config.Mode == "" {
		config.Mode = "text"
	}

	if config.Mode != "template" && config.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if config.Mode == "text" && config.Body == "" {
		return fmt.Errorf("body is required")
	}

	if config.Mode == "html" && strings.TrimSpace(config.HTMLBody) == "" {
		return fmt.Errorf("htmlBody is required")
	}

	if config.Mode == "template" && strings.TrimSpace(config.TemplateID) == "" {
		return fmt.Errorf("templateId is required")
	}

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
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("failed to decode configuration: %v", err)})
	}

	if config.To == "" {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: "to is required"})
	}

	if config.Mode == "" {
		config.Mode = "text"
	}

	if config.Mode != "template" && config.Subject == "" {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: "subject is required"})
	}

	if config.Mode == "text" && config.Body == "" {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: "body is required"})
	}

	if config.Mode == "html" && strings.TrimSpace(config.HTMLBody) == "" {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: "htmlBody is required"})
	}

	if config.Mode == "template" && strings.TrimSpace(config.TemplateID) == "" {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: "templateId is required"})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("failed to create SendGrid client: %v", err)})
	}

	toAddrs, err := parseEmailList(config.To)
	if err != nil {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("invalid 'to' email addresses: %v", err)})
	}

	ccAddrs, err := parseEmailList(config.Cc)
	if err != nil {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("invalid 'cc' email addresses: %v", err)})
	}

	bccAddrs, err := parseEmailList(config.Bcc)
	if err != nil {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("invalid 'bcc' email addresses: %v", err)})
	}

	fromName := config.FromName
	fromEmail := config.FromEmail
	if fromName == "" {
		fromName = optionalIntegrationConfig(ctx.Integration, "fromName")
	}
	if fromEmail == "" {
		fromEmail = optionalIntegrationConfig(ctx.Integration, "fromEmail")
	}

	if strings.TrimSpace(fromEmail) == "" {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: "fromEmail is required"})
	}

	if _, err := mail.ParseAddress(fromEmail); err != nil {
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("invalid 'fromEmail' address: %v", err)})
	}

	if config.ReplyTo != "" {
		if _, err := mail.ParseAddress(config.ReplyTo); err != nil {
			return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("invalid 'replyTo' address: %v", err)})
		}
	}

	personalization := Personalization{
		To:                  toEmailAddresses(toAddrs),
		Cc:                  toEmailAddresses(ccAddrs),
		Bcc:                 toEmailAddresses(bccAddrs),
		DynamicTemplateData: config.TemplateData,
	}

	request := MailSendRequest{
		Personalizations: []Personalization{personalization},
		From: EmailAddress{
			Email: fromEmail,
			Name:  fromName,
		},
		Subject:    config.Subject,
		TemplateID: config.TemplateID,
	}
	if categories := parseCSV(config.Categories); len(categories) > 0 {
		request.Categories = categories
	}

	if config.ReplyTo != "" {
		request.ReplyTo = &EmailAddress{Email: config.ReplyTo}
	}

	switch config.Mode {
	case "template":
		// Template drives content and subject.
	case "html":
		request.Content = []EmailContent{{Type: "text/html", Value: config.HTMLBody}}
	default:
		request.Content = []EmailContent{{Type: "text/plain", Value: config.Body}}
	}

	result, err := client.SendEmail(request)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return c.emitFailed(ctx, SendEmailFailure{
				Error:        apiErr.Error(),
				StatusCode:   apiErr.StatusCode,
				ResponseBody: apiErr.Body,
			})
		}
		return c.emitFailedAndFail(ctx, SendEmailFailure{Error: fmt.Sprintf("failed to send email: %v", err)})
	}

	payload := map[string]any{
		"messageId":  result.MessageID,
		"status":     result.Status,
		"statusCode": result.StatusCode,
		"to":         toAddrs,
		"subject":    config.Subject,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, SendEmailPayloadType, []any{payload})
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

func (c *SendEmail) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SendEmail) emitFailed(ctx core.ExecutionContext, payload SendEmailFailure) error {
	return ctx.ExecutionState.Emit(SendEmailFailedChannel, SendEmailFailedPayloadType, []any{payload})
}

func (c *SendEmail) emitFailedAndFail(ctx core.ExecutionContext, payload SendEmailFailure) error {
	if err := ctx.ExecutionState.Emit(SendEmailFailedChannel, SendEmailFailedPayloadType, []any{payload}); err != nil {
		return err
	}

	return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, payload.Error)
}

func toEmailAddresses(values []string) []EmailAddress {
	if len(values) == 0 {
		return nil
	}
	addresses := make([]EmailAddress, 0, len(values))
	for _, value := range values {
		addresses = append(addresses, EmailAddress{Email: value})
	}
	return addresses
}

func optionalIntegrationConfig(ctx core.IntegrationContext, key string) string {
	if ctx == nil {
		return ""
	}

	value, err := ctx.GetConfig(key)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(value))
}

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

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		result = append(result, item)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
