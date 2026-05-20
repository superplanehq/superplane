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

const DeleteJiraIncidentPayloadType = "jira.incident.deleted"

type DeleteIncident struct{}

type DeleteIncidentSpec struct {
	Project string `json:"project,omitempty"`
	Issue   string `json:"issue"`
}

func (c *DeleteIncident) Name() string {
	return "jira.deleteIncident"
}

func (c *DeleteIncident) Label() string {
	return "Delete Incident"
}

func (c *DeleteIncident) Description() string {
	return "Delete a Jira Service Management incident by issue id or issue key"
}

func (c *DeleteIncident) Documentation() string {
	return `The Delete Incident component removes an incident from Jira Service Management.

## Use Cases

- **Automated cleanup**: Remove incidents created during tests or erroneous automation
- **Lifecycle workflows**: Delete when a duplicate or mistaken incident is closed upstream

## Configuration

- **Project** (optional): Narrows the issue picker to one Jira project (up to 500 most recently updated issues in that project).
- **Issue**: The incident issue to delete (issue key from Jira search).

## Output

Confirms deletion with **deleted** set to true.`
}

func (c *DeleteIncident) Icon() string {
	return "jira"
}

func (c *DeleteIncident) Color() string {
	return "red"
}

func (c *DeleteIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional: limit the issue list to this Jira project",
			Placeholder: "Any project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Issue key (or id) of the incident to delete",
			Placeholder: "Select an issue",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "issue",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
}

func (c *DeleteIncident) Setup(ctx core.SetupContext) error {
	spec := DeleteIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if _, err := cloudIDFromIntegration(ctx.Integration); err != nil {
		return err
	}

	if strings.TrimSpace(spec.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	return nil
}

func (c *DeleteIncident) Execute(ctx core.ExecutionContext) error {
	spec := DeleteIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	issueID, err := client.ResolveNumericIssueID(spec.Issue)
	if err != nil {
		return err
	}

	if err := client.DeleteIncident(cloudID, issueID); err != nil {
		return fmt.Errorf("failed to delete incident: %w", err)
	}

	payload := map[string]any{
		"deleted": true,
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteJiraIncidentPayloadType,
		[]any{payload},
	)
}

func (c *DeleteIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
