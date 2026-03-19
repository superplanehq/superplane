package sendemail

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName = "sendEmail"
	PayloadType   = "sendEmail.sent"

	RecipientModeEmails  = "emails"
	RecipientModeMembers = "members"

	RecipientTypeUser  = "user"
	RecipientTypeRole  = "role"
	RecipientTypeGroup = "group"
)

func init() {
	registry.RegisterComponent(ComponentName, &SendEmail{})
}

type SendEmail struct{}

type Config struct {
	RecipientMode string      `json:"recipientMode" mapstructure:"recipientMode"`
	To            string      `json:"to" mapstructure:"to"`
	Recipients    []Recipient `json:"recipients" mapstructure:"recipients"`
	Subject       string      `json:"subject" mapstructure:"subject"`
	Body          string      `json:"body" mapstructure:"body"`
	URL           string      `json:"url" mapstructure:"url"`
	URLLabel      string      `json:"urlLabel" mapstructure:"urlLabel"`
}

type Recipient struct {
	Type  string `json:"type" mapstructure:"type"`
	User  string `json:"user" mapstructure:"user"`
	Role  string `json:"role" mapstructure:"role"`
	Group string `json:"group" mapstructure:"group"`
}

type OutputMetadata struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
}

func (c *SendEmail) Name() string {
	return ComponentName
}

func (c *SendEmail) Label() string {
	return "Send Email Notification"
}

func (c *SendEmail) Description() string {
	return "Send an email notification using the system email provider"
}

func (c *SendEmail) Documentation() string {
	return `The Send Email Notification component sends emails through the system's configured email provider (Resend or SMTP) without requiring a separate integration setup.

## Use Cases

- **Notifications**: Send email notifications for workflow events
- **Alerts**: Email alerts for errors or important conditions
- **Status updates**: Notify stakeholders about workflow progress
- **User communications**: Send emails to users as part of automated workflows

## Recipient Modes

### Email addresses
Specify one or more email addresses directly (comma-separated). Useful when sending to external recipients or specific known addresses.

### SuperPlane members
Select recipients from your organization's users, groups, or roles. The system resolves the actual email addresses at send time.

## Configuration

- **Recipient mode**: Choose between email addresses or SuperPlane members
- **To** (email mode): Comma-separated email addresses
- **Recipients** (members mode): List of users, groups, or roles
- **Subject**: Email subject line (supports expressions)
- **Body**: Email body content (supports expressions)
- **URL** (optional): A link to include in the email
- **URL Label** (optional): Label for the link button (defaults to "Open in SuperPlane")

## Output

Emits the list of recipients and the subject to the default output channel.`
}

func (c *SendEmail) Icon() string {
	return "mail"
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
			Name:     "recipientMode",
			Label:    "Send to",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  RecipientModeEmails,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Value: RecipientModeEmails, Label: "Email addresses"},
						{Value: RecipientModeMembers, Label: "SuperPlane members"},
					},
				},
			},
		},
		{
			Name:        "to",
			Label:       "To",
			Type:        configuration.FieldTypeString,
			Description: "Recipient email addresses (comma-separated for multiple)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "recipientMode", Values: []string{RecipientModeEmails}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "recipientMode", Values: []string{RecipientModeEmails}},
			},
		},
		{
			Name:        "recipients",
			Label:       "Recipients",
			Description: "Users, groups, or roles to send the email to",
			Type:        configuration.FieldTypeList,
			Default:     `[{"type":"user"}]`,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "recipientMode", Values: []string{RecipientModeMembers}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "recipientMode", Values: []string{RecipientModeMembers}},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Recipient",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Recipient type",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								Default:  RecipientTypeUser,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Value: RecipientTypeUser, Label: "Specific user"},
											{Value: RecipientTypeGroup, Label: "Group"},
											{Value: RecipientTypeRole, Label: "Role"},
										},
									},
								},
							},
							{
								Name:  "user",
								Label: "User",
								Type:  configuration.FieldTypeUser,
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "type", Values: []string{RecipientTypeUser}},
								},
							},
							{
								Name:  "role",
								Label: "Role",
								Type:  configuration.FieldTypeRole,
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "type", Values: []string{RecipientTypeRole}},
								},
							},
							{
								Name:  "group",
								Label: "Group",
								Type:  configuration.FieldTypeGroup,
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "type", Values: []string{RecipientTypeGroup}},
								},
							},
						},
					},
				},
			},
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
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional link to include in the email",
		},
		{
			Name:        "urlLabel",
			Label:       "URL Label",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Label for the link button (defaults to \"Open in SuperPlane\")",
		},
	}
}

func (c *SendEmail) Setup(ctx core.SetupContext) error {
	var config Config
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	switch config.RecipientMode {
	case RecipientModeEmails:
		if config.To == "" {
			return fmt.Errorf("to is required when sending to email addresses")
		}

		if _, err := parseEmailList(config.To); err != nil {
			return fmt.Errorf("invalid email addresses: %w", err)
		}

	case RecipientModeMembers:
		if len(config.Recipients) == 0 {
			return fmt.Errorf("at least one recipient is required when sending to members")
		}

		for i, r := range config.Recipients {
			switch r.Type {
			case RecipientTypeUser:
				if r.User == "" {
					return fmt.Errorf("recipient %d: user is required", i+1)
				}
			case RecipientTypeRole:
				if r.Role == "" {
					return fmt.Errorf("recipient %d: role is required", i+1)
				}
			case RecipientTypeGroup:
				if r.Group == "" {
					return fmt.Errorf("recipient %d: group is required", i+1)
				}
			default:
				return fmt.Errorf("recipient %d: unknown type %q", i+1, r.Type)
			}
		}

	default:
		return fmt.Errorf("unknown recipient mode %q", config.RecipientMode)
	}

	metadata := OutputMetadata{
		Subject: config.Subject,
	}

	if config.RecipientMode == RecipientModeEmails {
		emails, _ := parseEmailList(config.To)
		metadata.To = emails
	}

	return ctx.Metadata.Set(metadata)
}

func (c *SendEmail) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendEmail) Execute(ctx core.ExecutionContext) error {
	var config Config
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if config.Body == "" {
		return fmt.Errorf("body is required")
	}

	if ctx.Notifications == nil {
		return fmt.Errorf("notification context is not available")
	}

	if !ctx.Notifications.IsAvailable() {
		return fmt.Errorf("email delivery is not configured for this organization; configure SMTP settings or a Resend API key")
	}

	receivers, err := buildReceivers(config)
	if err != nil {
		return fmt.Errorf("failed to build recipients: %w", err)
	}

	if err := ctx.Notifications.Send(config.Subject, config.Body, config.URL, config.URLLabel, receivers); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	output := map[string]any{
		"subject": config.Subject,
		"to":      receivers.Emails,
		"groups":  receivers.Groups,
		"roles":   receivers.Roles,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{output},
	)
}

func (c *SendEmail) Actions() []core.Action {
	return []core.Action{}
}

func (c *SendEmail) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("sendEmail does not support actions")
}

func (c *SendEmail) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *SendEmail) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendEmail) Cleanup(ctx core.SetupContext) error {
	return nil
}

func buildReceivers(config Config) (core.NotificationReceivers, error) {
	receivers := core.NotificationReceivers{}

	switch config.RecipientMode {
	case RecipientModeEmails:
		emails, err := parseEmailList(config.To)
		if err != nil {
			return receivers, err
		}

		receivers.Emails = emails

	case RecipientModeMembers:
		emailSet := map[string]struct{}{}
		groupSet := map[string]struct{}{}
		roleSet := map[string]struct{}{}

		for _, r := range config.Recipients {
			switch r.Type {
			case RecipientTypeUser:
				if r.User != "" {
					emailSet[r.User] = struct{}{}
				}
			case RecipientTypeRole:
				if r.Role != "" {
					roleSet[r.Role] = struct{}{}
				}
			case RecipientTypeGroup:
				if r.Group != "" {
					groupSet[r.Group] = struct{}{}
				}
			}
		}

		receivers.Emails = mapKeys(emailSet)
		receivers.Groups = mapKeys(groupSet)
		receivers.Roles = mapKeys(roleSet)

	default:
		return receivers, fmt.Errorf("unknown recipient mode %q", config.RecipientMode)
	}

	return receivers, nil
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
			return nil, fmt.Errorf("invalid email address %q: %w", addr, err)
		}

		result = append(result, addr)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid email addresses found")
	}

	return result, nil
}

func mapKeys(input map[string]struct{}) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	return result
}
