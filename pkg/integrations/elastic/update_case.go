package elastic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateCase struct{}

type UpdateCaseConfiguration struct {
	CaseID      string   `json:"caseId" mapstructure:"caseId"`
	Version     string   `json:"version" mapstructure:"version"`
	Title       string   `json:"title" mapstructure:"title"`
	Description string   `json:"description" mapstructure:"description"`
	Status      string   `json:"status" mapstructure:"status"`
	Severity    string   `json:"severity" mapstructure:"severity"`
	Tags        []string `json:"tags" mapstructure:"tags"`
}

func (c *UpdateCase) Name() string  { return "elastic.updateCase" }
func (c *UpdateCase) Label() string { return "Update Case" }
func (c *UpdateCase) Description() string {
	return "Update an existing Kibana Security case"
}
func (c *UpdateCase) Icon() string  { return "elastic" }
func (c *UpdateCase) Color() string { return "gray" }

func (c *UpdateCase) Documentation() string {
	return `The Update Case component applies a partial update to an existing Kibana Security case.

## Configuration

- **Case ID**: The ID of the case to update
- **Version**: The current case version for optimistic locking (obtain from a preceding Get Case step)
- **Title**: New title for the case (optional)
- **Description**: New description for the case (optional)
- **Status**: New status for the case (optional)
- **Severity**: New severity for the case (optional)
- **Tags**: New tags for the case (optional)

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The case ID
- ` + "`title`" + `: The updated case title
- ` + "`status`" + `: The updated case status
- ` + "`severity`" + `: The updated case severity
- ` + "`version`" + `: The new case version`
}

func (c *UpdateCase) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateCase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "caseId",
			Label:       "Case ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the Kibana case to update.",
		},
		{
			Name:        "version",
			Label:       "Version",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The current case version for optimistic locking. Obtain from a preceding Get Case step.",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New title for the case.",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New description for the case.",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "New status for the case.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: "open"},
						{Label: "In Progress", Value: "in-progress"},
						{Label: "Closed", Value: "closed"},
					},
				},
			},
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "New severity for the case.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Low", Value: "low"},
						{Label: "Medium", Value: "medium"},
						{Label: "High", Value: "high"},
						{Label: "Critical", Value: "critical"},
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "New tags for the case.",
		},
	}
}

func (c *UpdateCase) Setup(ctx core.SetupContext) error {
	var config UpdateCaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.CaseID) == "" {
		return fmt.Errorf("caseId is required")
	}

	if strings.TrimSpace(config.Version) == "" {
		return fmt.Errorf("version is required")
	}

	if !hasCaseUpdates(config) {
		return fmt.Errorf("at least one field to update is required")
	}

	return nil
}

func (c *UpdateCase) Execute(ctx core.ExecutionContext) error {
	var config UpdateCaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.CaseID = strings.TrimSpace(config.CaseID)
	if config.CaseID == "" {
		return ctx.ExecutionState.Fail("error", "caseId is required")
	}

	config.Version = strings.TrimSpace(config.Version)
	if config.Version == "" {
		return ctx.ExecutionState.Fail("error", "version is required")
	}

	if !hasCaseUpdates(config) {
		return ctx.ExecutionState.Fail("error", "at least one field to update is required")
	}

	updates := map[string]any{}
	if v := strings.TrimSpace(config.Title); v != "" {
		updates["title"] = v
	}
	if v := strings.TrimSpace(config.Description); v != "" {
		updates["description"] = v
	}
	if v := strings.TrimSpace(config.Status); v != "" {
		updates["status"] = v
	}
	if v := strings.TrimSpace(config.Severity); v != "" {
		updates["severity"] = v
	}
	if config.Tags != nil {
		updates["tags"] = config.Tags
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create Elastic client: %v", err))
	}

	resp, err := client.UpdateCase(config.CaseID, config.Version, updates)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update case: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"elastic.case.updated",
		[]any{map[string]any{
			"id":        resp.ID,
			"title":     resp.Title,
			"status":    resp.Status,
			"severity":  resp.Severity,
			"version":   resp.Version,
			"updatedAt": resp.UpdatedAt,
		}},
	)
}

func (c *UpdateCase) Actions() []core.Action                  { return nil }
func (c *UpdateCase) HandleAction(_ core.ActionContext) error { return nil }
func (c *UpdateCase) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *UpdateCase) Cleanup(_ core.SetupContext) error       { return nil }
func (c *UpdateCase) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *UpdateCase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func hasCaseUpdates(config UpdateCaseConfiguration) bool {
	if strings.TrimSpace(config.Title) != "" {
		return true
	}
	if strings.TrimSpace(config.Description) != "" {
		return true
	}
	if strings.TrimSpace(config.Status) != "" {
		return true
	}
	if strings.TrimSpace(config.Severity) != "" {
		return true
	}

	return config.Tags != nil
}
