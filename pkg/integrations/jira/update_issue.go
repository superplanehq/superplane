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

const UpdateIssuePayloadType = "jira.issue"

type UpdateIssue struct{}

type UpdateIssueSpec struct {
	Project     string    `json:"project" mapstructure:"project"`
	IssueKey    string    `json:"issueKey" mapstructure:"issueKey"`
	Summary     *string   `json:"summary,omitempty" mapstructure:"summary"`
	Description *string   `json:"description,omitempty" mapstructure:"description"`
	IssueType   *string   `json:"issueType,omitempty" mapstructure:"issueType"`
	Assignee    *string   `json:"assignee,omitempty" mapstructure:"assignee"`
	Priority    *string   `json:"priority,omitempty" mapstructure:"priority"`
	Labels      *[]string `json:"labels,omitempty" mapstructure:"labels"`
	NotifyUsers *bool     `json:"notifyUsers,omitempty" mapstructure:"notifyUsers"`
}

func (c *UpdateIssue) Name() string {
	return "jira.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update fields on a Jira issue"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component updates fields on an existing Jira issue.

## Use Cases

- **Automated triage**: change status, priority, or assignee when a workflow processes the issue
- **Cross-tool sync**: mirror state from another system into Jira
- **Bulk relabeling**: apply labels based on workflow inputs

## Configuration

- **Project**: The Jira project the issue belongs to
- **Issue Key**: The issue key (e.g. ` + "`PROJ-123`" + `)
- **Summary**, **Description**, **Issue Type**, **Assignee**, **Priority**, **Labels**: Optional fields to update. At least one must be supplied.
- **Notify Users**: Whether to send notification emails (defaults to Jira's behaviour)

## Output

Returns the updated Jira issue as fetched after the update.`
}

func (c *UpdateIssue) Icon() string {
	return "jira"
}

func (c *UpdateIssue) Color() string {
	return "blue"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Jira project the issue belongs to",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue key (e.g. PROJ-123)",
			Placeholder: "PROJ-123",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New issue summary",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New issue description (plain text; will be wrapped in ADF)",
		},
		{
			Name:        "issueType",
			Label:       "Issue Type",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "New issue type (scoped to the project)",
			Placeholder: "Select an issue type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "issueType",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:        "assignee",
			Label:       "Assignee",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "User to assign the issue to (leave empty to keep the current assignee)",
			Placeholder: "Select a user",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "assignee",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Priority (instance-level, applies to all projects)",
			Placeholder: "Select a priority",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "priority",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Replace the issue labels with the given list",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
					ItemLabel: "Label",
				},
			},
		},
		{
			Name:        "notifyUsers",
			Label:       "Notify Users",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Description: "Send notification emails to watchers (Jira default is true)",
			Default:     true,
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	spec := UpdateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Project) == "" {
		return fmt.Errorf("project is required")
	}

	if strings.TrimSpace(spec.IssueKey) == "" {
		return fmt.Errorf("issueKey is required")
	}

	if !hasAnyUpdate(spec) {
		return fmt.Errorf("at least one field to update must be provided")
	}

	project, err := requireProject(ctx.HTTP, ctx.Integration, spec.Project)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Project: project})
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueKey := strings.TrimSpace(spec.IssueKey)
	if issueKey == "" {
		return fmt.Errorf("issueKey is required")
	}

	if !hasAnyUpdate(spec) {
		return fmt.Errorf("at least one field to update must be provided")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	req := &UpdateIssueRequest{Fields: buildUpdateFields(spec)}

	if err := client.UpdateIssue(issueKey, req, UpdateIssueOptions{NotifyUsers: spec.NotifyUsers}); err != nil {
		return fmt.Errorf("failed to update issue: %v", err)
	}

	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch updated issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateIssuePayloadType,
		[]any{issue},
	)
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func hasAnyUpdate(spec UpdateIssueSpec) bool {
	return spec.Summary != nil ||
		spec.Description != nil ||
		spec.IssueType != nil ||
		spec.Assignee != nil ||
		spec.Priority != nil ||
		spec.Labels != nil
}

func buildUpdateFields(spec UpdateIssueSpec) map[string]any {
	fields := map[string]any{}

	if spec.Summary != nil {
		fields["summary"] = *spec.Summary
	}

	if spec.Description != nil {
		if *spec.Description == "" {
			fields["description"] = nil
		} else {
			fields["description"] = WrapInADF(*spec.Description)
		}
	}

	if spec.IssueType != nil {
		fields["issuetype"] = map[string]any{"name": *spec.IssueType}
	}

	if spec.Assignee != nil {
		accountID := *spec.Assignee
		if accountID == "" || accountID == "-1" {
			fields["assignee"] = nil
		} else {
			fields["assignee"] = map[string]any{"accountId": accountID}
		}
	}

	if spec.Priority != nil {
		fields["priority"] = map[string]any{"name": *spec.Priority}
	}

	if spec.Labels != nil {
		labels := []string{}
		if *spec.Labels != nil {
			labels = append(labels, *spec.Labels...)
		}
		fields["labels"] = labels
	}

	return fields
}
