package sendemail

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName = "sendEmail"
	PayloadType   = "sendEmail.sent"

	RecipientTypeUser  = "user"
	RecipientTypeRole  = "role"
	RecipientTypeGroup = "group"
)

func init() {
	registry.RegisterComponent(ComponentName, &SendEmail{})
}

type SendEmail struct{}

type Config struct {
	Recipients []Recipient `json:"recipients" mapstructure:"recipients"`
	Subject    string      `json:"subject" mapstructure:"subject"`
	Body       string      `json:"body" mapstructure:"body"`
}

type Recipient struct {
	Type  string `json:"type" mapstructure:"type"`
	User  string `json:"user" mapstructure:"user"`
	Role  string `json:"role" mapstructure:"role"`
	Group string `json:"group" mapstructure:"group"`
}

type OutputMetadata struct {
	Subject string `json:"subject"`
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

## Recipients

Select recipients from your organization's users, groups, or roles. The system resolves the actual email addresses at send time.

## Configuration

- **Recipients**: List of users, groups, or roles
- **Subject**: Email subject line (supports expressions)
- **Body**: Email body content (supports expressions)

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
			Name:        "recipients",
			Label:       "Recipients",
			Description: "Users, groups, or roles to send the email to",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     `[{"type":"user"}]`,
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

	if len(config.Recipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
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

	metadata := OutputMetadata{
		Subject: config.Subject,
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

	receivers, err := buildReceivers(config, ctx.Auth)
	if err != nil {
		return fmt.Errorf("failed to build recipients: %w", err)
	}

	if err := ctx.Notifications.Send(config.Subject, config.Body, "", "", receivers); err != nil {
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

func buildReceivers(config Config, auth core.AuthContext) (core.NotificationReceivers, error) {
	emailSet := map[string]struct{}{}
	groupSet := map[string]struct{}{}
	roleSet := map[string]struct{}{}

	for _, r := range config.Recipients {
		switch r.Type {
		case RecipientTypeUser:
			if err := resolveUserEmail(r.User, auth, emailSet); err != nil {
				return core.NotificationReceivers{}, err
			}
		case RecipientTypeRole:
			addIfNotEmpty(r.Role, roleSet)
		case RecipientTypeGroup:
			addIfNotEmpty(r.Group, groupSet)
		}
	}

	return core.NotificationReceivers{
		Emails: mapKeys(emailSet),
		Groups: mapKeys(groupSet),
		Roles:  mapKeys(roleSet),
	}, nil
}

func resolveUserEmail(rawID string, auth core.AuthContext, dest map[string]struct{}) error {
	if rawID == "" {
		return nil
	}

	userID, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("invalid user ID %q: %w", rawID, err)
	}

	user, err := auth.GetUser(userID)
	if err != nil {
		return fmt.Errorf("failed to resolve user %q: %w", rawID, err)
	}

	if user.Email != "" {
		dest[user.Email] = struct{}{}
	}

	return nil
}

func addIfNotEmpty(value string, dest map[string]struct{}) {
	if value != "" {
		dest[value] = struct{}{}
	}
}

func mapKeys(input map[string]struct{}) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	return result
}
