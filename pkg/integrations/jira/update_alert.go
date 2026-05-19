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
	Alert            string         `json:"alert" mapstructure:"alert"`
	Description      string         `json:"description,omitempty" mapstructure:"description"`
	Message          string         `json:"message,omitempty" mapstructure:"message"`
	Priority         string         `json:"priority,omitempty" mapstructure:"priority"`
	Assignee         string         `json:"assignee,omitempty" mapstructure:"assignee"`
	NewNote          string         `json:"newNote,omitempty" mapstructure:"newNote"`
	PatchNote        map[string]any `json:"patchExistingNote,omitempty" mapstructure:"patchExistingNote"`
	AcknowledgeAlert bool           `json:"acknowledgeAlert,omitempty" mapstructure:"acknowledgeAlert"`
	CloseAlert       bool           `json:"closeAlert,omitempty" mapstructure:"closeAlert"`
}

func (c *UpdateAlert) Name() string {
	return "jira.updateAlert"
}

func (c *UpdateAlert) Label() string {
	return "Update Alert"
}

func (c *UpdateAlert) Description() string {
	return "Update, assign, add a note to, acknowledge, or close a Jira Service Management Ops alert"
}

func (c *UpdateAlert) Documentation() string {
	return `The Update Alert component runs updates on alerts on Jira Service Management.

Toggle each optional subsection you need; untouched sections are not sent to Jira.

## Optional updates

- **Description** → updates alert description  
- **Message** → updates alert message  
- **Priority** → updates alert priority (omit with "Don't set")  
- **Assignment** → assigns the alert to a Jira user (Atlassian account ID from the assignee picker)
- **New note** → adds a new note  
- **Update existing note** → updates existing note with nested **note id** + **note** text  
- **Acknowledge alert** → acknowledges alert  
- **Close alert** → closes alert

After updates, SuperPlane polls the **last asynchronous** Ops response (when applicable) and emits a fresh **GET alert** payload like **Get Alert**.`
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

func visWhen(field string) []configuration.VisibilityCondition {
	return []configuration.VisibilityCondition{{Field: field, Values: []string{"true"}}}
}

func (c *UpdateAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alert",
			Label:       "Alert",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Ops alerts recently returned by List alerts (refresh the picker after new alerts appear)",
			Placeholder: "Select an alert",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "alert",
				},
			},
		},
		{
			Name:        "setDescription",
			Label:       "Change description",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enables updating the Ops alert description",
		},
		{
			Name:                 "description",
			Label:                "Description",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			Description:          "New description text applied when Change description is on",
			VisibilityConditions: visWhen("setDescription"),
		},
		{
			Name:        "setMessage",
			Label:       "Change message",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enables updating the alert message",
		},
		{
			Name:                 "message",
			Label:                "Message",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Description:          "Replacement message applied when Change message is on",
			VisibilityConditions: visWhen("setMessage"),
		},
		{
			Name:        "setPriority",
			Label:       "Change priority",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enables updating Ops alert priority",
		},
		{
			Name:                 "priority",
			Label:                "Priority",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              "__none__",
			Description:          "Priority applied when Change priority is on",
			VisibilityConditions: visWhen("setPriority"),
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
			Name:        "setAssignment",
			Label:       "Assign alert",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Enable assigning the Ops alert via the assign API",
		},
		{
			Name:                 "assignee",
			Label:                "Assignee",
			Type:                 configuration.FieldTypeIntegrationResource,
			Required:             false,
			Description:          "User to assign the alert to when Assign alert is enabled",
			Placeholder:          "Select a user",
			VisibilityConditions: visWhen("setAssignment"),
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "assignee",
				},
			},
		},
		{
			Name:        "newNote",
			Label:       "New note",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "When enabled, creates a note with this text",
		},
		{
			Name:        "patchExistingNote",
			Label:       "Update existing note",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Default:     map[string]any{"noteId": "", "note": ""},
			Description: "When enabled, updates the note identified by note id using the provided text",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:        "noteId",
							Label:       "Note ID",
							Type:        configuration.FieldTypeString,
							Required:    true,
							Description: "Id of the Ops alert note to update",
						},
						{
							Name:        "note",
							Label:       "Note",
							Type:        configuration.FieldTypeText,
							Required:    true,
							Description: "Replacement note text",
						},
					},
				},
			},
		},
		{
			Name:        "acknowledgeAlert",
			Label:       "Acknowledge alert",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Togglable:   true,
			Description: "When enabled, acknowledges the alert before other async steps complete",
		},
		{
			Name:        "closeAlert",
			Label:       "Close alert",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Togglable:   true,
			Description: "When enabled, closes an alert",
		},
	}
}

func (c *UpdateAlert) Setup(ctx core.SetupContext) error {
	spec := UpdateAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}
	alertKey := strings.TrimSpace(spec.Alert)
	if alertKey == "" {
		return fmt.Errorf("alert is required")
	}
	cfg := ConfigurationAsSliceMap(ctx.Configuration)

	if err := validateUpdateAlertConfigurable(cfg, spec); err != nil {
		return err
	}

	summaries := buildUpdateAlertConfiguredSummaries(cfg, spec)
	meta := UpdateAlertNodeMetadata{UpdateSummaries: summaries}

	if ctx.HTTP != nil {
		client, cerr := NewClient(ctx.HTTP, ctx.Integration)
		if cerr == nil {
			if row, gerr := client.GetOpsAlert(cloudID, alertKey); gerr == nil {
				meta.AlertLabel = opsAlertIntegrationResourceLabel(row, alertKey)
			}
		}
	}

	return ctx.Metadata.Set(meta)
}

func validateUpdateAlertConfigurable(cfg map[string]any, spec UpdateAlertSpec) error {
	count := len(buildUpdateAlertConfiguredSummaries(cfg, spec))
	if count == 0 {
		return fmt.Errorf(
			"enable at least one update: description, message, priority, assign, new note, existing note patch, acknowledge, or close",
		)
	}

	if isTruthy(cfg, "setDescription") {
		if strings.TrimSpace(spec.Description) == "" {
			return fmt.Errorf("description cannot be empty when Change description is enabled")
		}
	}
	if isTruthy(cfg, "setMessage") {
		if strings.TrimSpace(spec.Message) == "" {
			return fmt.Errorf("message cannot be empty when Change message is enabled")
		}
	}
	if isTruthy(cfg, "setPriority") {
		p := strings.TrimSpace(spec.Priority)
		if p == "" || p == "__none__" {
			return fmt.Errorf(`choose a concrete priority level or disable "Change priority"`)
		}
	}

	if isTruthy(cfg, "setAssignment") {
		if strings.TrimSpace(spec.Assignee) == "" {
			return fmt.Errorf("assignee is required when Assign alert is enabled")
		}
	}

	if cfgSectionEnabled(cfg, "newNote") {
		if strings.TrimSpace(spec.NewNote) == "" {
			return fmt.Errorf("new note cannot be empty when enabled")
		}
	}

	if cfgSectionEnabled(cfg, "patchExistingNote") {
		sub := ConfigurationAsSliceMap(spec.PatchNote)
		if len(sub) == 0 {
			return fmt.Errorf("update existing note requires note id and text when enabled")
		}
		noteID := strings.TrimSpace(opsAlertStringField(sub, "noteId"))
		body := strings.TrimSpace(opsAlertStringField(sub, "note"))
		if noteID == "" || body == "" {
			return fmt.Errorf("update existing note requires both Note ID and Note text when enabled")
		}
	}

	return nil
}

func isTruthy(cfg map[string]any, key string) bool {
	raw, ok := cfg[key]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func cfgSectionEnabled(cfg map[string]any, key string) bool {
	if cfg == nil {
		return false
	}
	raw, ok := cfg[key]
	return ok && raw != nil
}

func buildUpdateAlertConfiguredSummaries(cfg map[string]any, spec UpdateAlertSpec) []string {
	if cfg == nil {
		return nil
	}
	var summaries []string
	if isTruthy(cfg, "setDescription") {
		summaries = append(summaries, "Description update")
	}
	if isTruthy(cfg, "setMessage") {
		summaries = append(summaries, "Message update")
	}
	if isTruthy(cfg, "setPriority") {
		p := strings.TrimSpace(spec.Priority)
		if p != "" && p != "__none__" {
			summaries = append(summaries, fmt.Sprintf("Priority → %s", p))
		} else {
			summaries = append(summaries, "Priority update")
		}
	}
	if isTruthy(cfg, "setAssignment") {
		summaries = append(summaries, "Assign user")
	}
	if cfgSectionEnabled(cfg, "newNote") {
		summaries = append(summaries, "Add note")
	}
	if cfgSectionEnabled(cfg, "patchExistingNote") {
		sub := ConfigurationAsSliceMap(spec.PatchNote)
		if len(sub) > 0 {
			id := strings.TrimSpace(opsAlertStringField(sub, "noteId"))
			if id != "" {
				summaries = append(summaries, fmt.Sprintf("Note patch (%s)", id))
			} else {
				summaries = append(summaries, "Existing note patch")
			}
		}
	}
	if cfgSectionEnabled(cfg, "acknowledgeAlert") && spec.AcknowledgeAlert {
		summaries = append(summaries, "Acknowledge")
	}
	if cfgSectionEnabled(cfg, "closeAlert") && spec.CloseAlert {
		summaries = append(summaries, "Close")
	}
	return summaries
}

func (c *UpdateAlert) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cfg := ConfigurationAsSliceMap(ctx.Configuration)
	if err := validateUpdateAlertConfigurable(cfg, spec); err != nil {
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
	alertKey := strings.TrimSpace(spec.Alert)

	var lastAsync *OpsAsyncSuccessResponse

	applyAsync := func(r *OpsAsyncSuccessResponse, callErr error) error {
		if callErr != nil {
			return callErr
		}
		if r != nil && strings.TrimSpace(r.RequestID) != "" {
			lastAsync = r
		}
		return nil
	}

	if isTruthy(cfg, "setDescription") {
		r, err := client.PatchOpsAlertDescription(cloudID, alertKey, strings.TrimSpace(spec.Description))
		if err := applyAsync(r, err); err != nil {
			return fmt.Errorf("patch description: %w", err)
		}
	}
	if isTruthy(cfg, "setMessage") {
		r, err := client.PatchOpsAlertMessage(cloudID, alertKey, strings.TrimSpace(spec.Message))
		if err := applyAsync(r, err); err != nil {
			return fmt.Errorf("patch message: %w", err)
		}
	}
	if isTruthy(cfg, "setPriority") {
		p := strings.TrimSpace(spec.Priority)
		if p != "" && p != "__none__" {
			r, err := client.PatchOpsAlertPriority(cloudID, alertKey, p)
			if err := applyAsync(r, err); err != nil {
				return fmt.Errorf("patch priority: %w", err)
			}
		}
	}
	if isTruthy(cfg, "setAssignment") {
		aid := strings.TrimSpace(spec.Assignee)
		if aid != "" {
			r, err := client.AssignOpsAlert(cloudID, alertKey, aid)
			if err := applyAsync(r, err); err != nil {
				return fmt.Errorf("assign alert: %w", err)
			}
		}
	}
	if cfgSectionEnabled(cfg, "newNote") {
		noteResp, noteErr := client.AddOpsAlertNote(cloudID, alertKey, strings.TrimSpace(spec.NewNote))
		if noteErr != nil {
			return fmt.Errorf("add note: %w", noteErr)
		}
		_ = noteResp // synchronous 200; does not contribute to polling
	}
	if cfgSectionEnabled(cfg, "patchExistingNote") {
		sub := ConfigurationAsSliceMap(spec.PatchNote)
		nid := strings.TrimSpace(opsAlertStringField(sub, "noteId"))
		body := strings.TrimSpace(opsAlertStringField(sub, "note"))
		if nid != "" && body != "" {
			noteResp, noteErr := client.PatchOpsAlertNote(cloudID, alertKey, nid, body)
			if noteErr != nil {
				return fmt.Errorf("patch note: %w", noteErr)
			}
			_ = noteResp
		}
	}
	if cfgSectionEnabled(cfg, "acknowledgeAlert") && spec.AcknowledgeAlert {
		r, err := client.AcknowledgeOpsAlert(cloudID, alertKey)
		if err := applyAsync(r, err); err != nil {
			return fmt.Errorf("acknowledge: %w", err)
		}
	}
	if cfgSectionEnabled(cfg, "closeAlert") && spec.CloseAlert {
		r, err := client.CloseOpsAlert(cloudID, alertKey)
		if err := applyAsync(r, err); err != nil {
			return fmt.Errorf("close: %w", err)
		}
	}

	pollReqID := ""
	if lastAsync != nil {
		pollReqID = lastAsync.RequestID
	}

	if pollReqID != "" {
		if _, err := client.ResolveAlertIDAfterOpsRequest(cloudID, pollReqID, alertKey); err != nil {
			return fmt.Errorf("wait for async Ops processing: %w", err)
		}
	}

	fresh, err := client.GetOpsAlert(cloudID, alertKey)
	if err != nil {
		return fmt.Errorf("failed to reload alert after updates: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateJiraAlertPayloadType,
		[]any{fresh},
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
