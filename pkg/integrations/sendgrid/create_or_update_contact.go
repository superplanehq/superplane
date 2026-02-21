package sendgrid

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	UpsertContactPayloadType       = "sendgrid.contact.upserted"
	UpsertContactFailedPayloadType = "sendgrid.contact.failed"
	UpsertContactFailedChannel     = "failed"
)

type CreateOrUpdateContact struct{}

type CreateOrUpdateContactConfiguration struct {
	Email        string         `json:"email" mapstructure:"email"`
	FirstName    string         `json:"firstName" mapstructure:"firstName"`
	LastName     string         `json:"lastName" mapstructure:"lastName"`
	ListIDs      []string       `json:"listIds" mapstructure:"listIds"`
	CustomFields map[string]any `json:"customFields" mapstructure:"customFields"`
}

type CreateOrUpdateContactMetadata struct {
	Email string `json:"email" mapstructure:"email"`
}

type CreateOrUpdateContactFailure struct {
	Error        string `json:"error"`
	StatusCode   int    `json:"statusCode,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
}

var expressionPlaceholderRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

func (c *CreateOrUpdateContact) Name() string {
	return "sendgrid.createOrUpdateContact"
}

func (c *CreateOrUpdateContact) Label() string {
	return "Create or Update Contact"
}

func (c *CreateOrUpdateContact) Description() string {
	return "Create or update a SendGrid contact"
}

func (c *CreateOrUpdateContact) Documentation() string {
	return `Create or update a contact in SendGrid using the Marketing Contacts API.

## Use Cases

- **Signup sync**: Add new signups to SendGrid lists for onboarding or newsletters
- **CRM sync**: Keep SendGrid contacts updated from your CRM or database
- **Post-purchase follow-up**: Add buyers to follow-up campaigns

## Configuration

- **Email**: Contact email address (unique identifier)
- **First Name**: Optional contact first name
- **Last Name**: Optional contact last name
- **List IDs**: Optional SendGrid list IDs to add the contact to
- **Custom Fields**: Optional custom fields map (must exist in SendGrid)

## Output Channels

- **Default**: Emitted when SendGrid accepts the upsert request
- **Failed**: Emitted when validation fails or the API request is rejected

## Notes

- Requires a SendGrid API key with Marketing Contacts permissions`
}

func (c *CreateOrUpdateContact) Icon() string {
	return "users"
}

func (c *CreateOrUpdateContact) Color() string {
	return "gray"
}

func (c *CreateOrUpdateContact) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  core.DefaultOutputChannel.Name,
			Label: "Success",
		},
		{
			Name:  UpsertContactFailedChannel,
			Label: "Failed",
		},
	}
}

func (c *CreateOrUpdateContact) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "email",
			Label:       "Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Contact email address",
		},
		{
			Name:        "firstName",
			Label:       "First Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Contact first name",
		},
		{
			Name:        "lastName",
			Label:       "Last Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Contact last name",
		},
		{
			Name:        "listIds",
			Label:       "List IDs",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "SendGrid list IDs to add the contact to",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "List ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "customFields",
			Label:       "Custom Fields",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Description: "Key-value pairs for custom fields (msut be predefined in SendGrid contact custom fields)",
		},
	}
}

func (c *CreateOrUpdateContact) Setup(ctx core.SetupContext) error {
	var config CreateOrUpdateContactConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Email) == "" {
		return fmt.Errorf("email is required")
	}

	if !expressionPlaceholderRegex.MatchString(config.Email) {
		if _, err := mail.ParseAddress(config.Email); err != nil {
			return fmt.Errorf("invalid 'email' address: %w", err)
		}
	}

	metadata := CreateOrUpdateContactMetadata{Email: config.Email}
	return ctx.Metadata.Set(metadata)
}

func (c *CreateOrUpdateContact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateOrUpdateContact) Execute(ctx core.ExecutionContext) error {
	var config CreateOrUpdateContactConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return c.emitFailedAndFail(ctx, CreateOrUpdateContactFailure{Error: fmt.Sprintf("failed to decode configuration: %v", err)})
	}

	if strings.TrimSpace(config.Email) == "" {
		return c.emitFailedAndFail(ctx, CreateOrUpdateContactFailure{Error: "email is required"})
	}

	if _, err := mail.ParseAddress(config.Email); err != nil {
		return c.emitFailedAndFail(ctx, CreateOrUpdateContactFailure{Error: fmt.Sprintf("invalid 'email' address: %v", err)})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return c.emitFailedAndFail(ctx, CreateOrUpdateContactFailure{Error: fmt.Sprintf("failed to create SendGrid client: %v", err)})
	}

	request := UpsertContactsRequest{
		Contacts: []ContactInput{
			{
				Email:        config.Email,
				FirstName:    strings.TrimSpace(config.FirstName),
				LastName:     strings.TrimSpace(config.LastName),
				CustomFields: config.CustomFields,
			},
		},
		ListIDs: filterListIDs(config.ListIDs),
	}

	result, err := client.UpsertContact(request)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			return c.emitFailed(ctx, CreateOrUpdateContactFailure{
				Error:        apiErr.Error(),
				StatusCode:   apiErr.StatusCode,
				ResponseBody: apiErr.Body,
			})
		}
		return c.emitFailedAndFail(ctx, CreateOrUpdateContactFailure{Error: fmt.Sprintf("failed to upsert contact: %v", err)})
	}

	payload := map[string]any{
		"jobId":      result.JobID,
		"status":     result.Status,
		"statusCode": result.StatusCode,
		"email":      config.Email,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UpsertContactPayloadType, []any{payload})
}

func (c *CreateOrUpdateContact) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateOrUpdateContact) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateOrUpdateContact) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateOrUpdateContact) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateOrUpdateContact) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateOrUpdateContact) emitFailed(ctx core.ExecutionContext, payload CreateOrUpdateContactFailure) error {
	return ctx.ExecutionState.Emit(UpsertContactFailedChannel, UpsertContactFailedPayloadType, []any{payload})
}

func (c *CreateOrUpdateContact) emitFailedAndFail(ctx core.ExecutionContext, payload CreateOrUpdateContactFailure) error {
	if err := ctx.ExecutionState.Emit(UpsertContactFailedChannel, UpsertContactFailedPayloadType, []any{payload}); err != nil {
		return err
	}

	return ctx.ExecutionState.Fail(models.CanvasNodeExecutionResultReasonError, payload.Error)
}

func filterListIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
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
