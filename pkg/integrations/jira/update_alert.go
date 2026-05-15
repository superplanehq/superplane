package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateJiraAlertPayloadType = "jira.alert.updated"

type UpdateAlert struct{}

// UpdateAlertSpec applies one or more optional Ops alert mutations in a single workflow step.
type UpdateAlertSpec struct {
	AlertID string `json:"alertId" mapstructure:"alertId"`

	Description string `json:"description,omitempty" mapstructure:"description"`
	Message     string `json:"message,omitempty" mapstructure:"message"`
	Priority    string `json:"priority,omitempty" mapstructure:"priority"`

	NewNote        string `json:"newNote,omitempty" mapstructure:"newNote"`
	ExistingNoteID string `json:"existingNoteId,omitempty" mapstructure:"existingNoteId"`
	ExistingNote   string `json:"existingNote,omitempty" mapstructure:"existingNote"`

	AcknowledgeAlert bool `json:"acknowledgeAlert,omitempty" mapstructure:"acknowledgeAlert"`
	CloseAlert       bool `json:"closeAlert,omitempty" mapstructure:"closeAlert"`
}

func (c *UpdateAlert) Name() string {
	return "jira.updateAlert"
}

func (c *UpdateAlert) Label() string {
	return "Update Alert"
}

func (c *UpdateAlert) Description() string {
	return "Update, add a note to, acknowledge, or close a Jira Service Management Ops alert"
}

func (c *UpdateAlert) Documentation() string {
	return `The Update Alert component runs one or more [Jira Service Management Ops Alerts API](https://developer.atlassian.com/cloud/jira/service-desk-ops/rest/v2/api-group-alerts/) operations on the same alert.

All fields except **Alert id** are optional; choose any combination your workflow needs.

## Optional mutations

- **Description** → PATCH alert description  
- **Message** → PATCH alert message  
- **Priority** → PATCH priority (omit with "Don't set")  
- **New note** → POST a new note  
- **Existing note id** + **Existing note** → PATCH that note  
- **Acknowledge alert** → POST acknowledge  
- **Close alert** → POST close

Mutations run in the order above. Many endpoints return asynchronously (HTTP 202); the emitted payload lists what was invoked and copies of API acknowledgement objects when returned.`
}

func (c *UpdateAlert) Icon() string {
	return "jira"
}

func (c *UpdateAlert) Color() string {
	return "purple"
}

func (c *UpdateAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alertId",
			Label:       "Alert ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jira Ops alert id",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "When set, PATCH the alert description to this text",
		},
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "When set, PATCH the alert message",
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "__none__",
			Description: "When set, PATCH alert priority",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Don't set", Value: "__none__"},
						{Label: "P1", Value: "P1"},
						{Label: "P2", Value: "P2"},
						{Label: "P3", Value: "P3"},
						{Label: "P4", Value: "P4"},
						{Label: "P5", Value: "P5"},
					},
				},
			},
		},
		{
			Name:        "newNote",
			Label:       "New note",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "When set, POST a new note with this text",
		},
		{
			Name:        "existingNoteId",
			Label:       "Existing note ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "When updating a note, the note id from Jira (use with Existing note)",
		},
		{
			Name:        "existingNote",
			Label:       "Existing note",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "New text when patching the note identified by Existing note ID",
		},
		{
			Name:        "acknowledgeAlert",
			Label:       "Acknowledge alert",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "When enabled, POST acknowledge on this alert",
		},
		{
			Name:        "closeAlert",
			Label:       "Close alert",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "When enabled, POST close on this alert",
		},
	}
}

func (c *UpdateAlert) Setup(ctx core.SetupContext) error {
	spec := UpdateAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if _, err := cloudIDFromIntegration(ctx.Integration); err != nil {
		return err
	}
	if strings.TrimSpace(spec.AlertID) == "" {
		return fmt.Errorf("alertId is required")
	}
	noteID := strings.TrimSpace(spec.ExistingNoteID)
	noteBody := strings.TrimSpace(spec.ExistingNote)
	switch {
	case noteID != "" && noteBody == "":
		return fmt.Errorf("existingNote is required when existingNoteId is set")
	case noteBody != "" && noteID == "":
		return fmt.Errorf("existingNoteId is required when updating an existing note; use New note to append")
	}
	if err := validateUpdateAlertHasOperations(spec); err != nil {
		return err
	}
	return nil
}

func validateUpdateAlertHasOperations(spec UpdateAlertSpec) error {
	hasPriority := strings.TrimSpace(spec.Priority) != "" && strings.TrimSpace(spec.Priority) != "__none__"
	if strings.TrimSpace(spec.Description) != "" ||
		strings.TrimSpace(spec.Message) != "" ||
		hasPriority ||
		strings.TrimSpace(spec.NewNote) != "" ||
		(strings.TrimSpace(spec.ExistingNoteID) != "" && strings.TrimSpace(spec.ExistingNote) != "") ||
		spec.AcknowledgeAlert ||
		spec.CloseAlert {
		return nil
	}
	return fmt.Errorf("choose at least one update: description, message, priority, new note, note update, acknowledge, or close")
}

func (c *UpdateAlert) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if err := validateUpdateAlertHasOperations(spec); err != nil {
		return err
	}
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	alertID := strings.TrimSpace(spec.AlertID)

	var operations []string
	result := map[string]any{"alertId": alertID}

	if s := strings.TrimSpace(spec.Description); s != "" {
		r, err := client.PatchOpsAlertDescription(cloudID, alertID, s)
		if err != nil {
			return fmt.Errorf("patch description: %w", err)
		}
		operations = append(operations, "patchDescription")
		result["patchDescription"] = r
	}
	if s := strings.TrimSpace(spec.Message); s != "" {
		r, err := client.PatchOpsAlertMessage(cloudID, alertID, s)
		if err != nil {
			return fmt.Errorf("patch message: %w", err)
		}
		operations = append(operations, "patchMessage")
		result["patchMessage"] = r
	}
	if p := strings.TrimSpace(spec.Priority); p != "" && p != "__none__" {
		r, err := client.PatchOpsAlertPriority(cloudID, alertID, p)
		if err != nil {
			return fmt.Errorf("patch priority: %w", err)
		}
		operations = append(operations, "patchPriority")
		result["patchPriority"] = r
	}
	if s := strings.TrimSpace(spec.NewNote); s != "" {
		note, err := client.AddOpsAlertNote(cloudID, alertID, s)
		if err != nil {
			return fmt.Errorf("add note: %w", err)
		}
		operations = append(operations, "addNote")
		result["addNote"] = note
	}
	if nid := strings.TrimSpace(spec.ExistingNoteID); nid != "" {
		note, err := client.PatchOpsAlertNote(cloudID, alertID, nid, strings.TrimSpace(spec.ExistingNote))
		if err != nil {
			return fmt.Errorf("patch note: %w", err)
		}
		operations = append(operations, "patchNote")
		result["patchNote"] = note
	}
	if spec.AcknowledgeAlert {
		r, err := client.AcknowledgeOpsAlert(cloudID, alertID)
		if err != nil {
			return fmt.Errorf("acknowledge: %w", err)
		}
		operations = append(operations, "acknowledge")
		result["acknowledge"] = r
	}
	if spec.CloseAlert {
		r, err := client.CloseOpsAlert(cloudID, alertID)
		if err != nil {
			return fmt.Errorf("close: %w", err)
		}
		operations = append(operations, "close")
		result["close"] = r
	}

	result["operations"] = operations
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateJiraAlertPayloadType,
		[]any{result},
	)
}

func (c *UpdateAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateAlert) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
