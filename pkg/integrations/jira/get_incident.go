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

const GetJiraIncidentPayloadType = "jira.incident.fetched"

type GetIncident struct{}

type GetIncidentSpec struct {
	Project string `json:"project,omitempty"`
	Issue   string `json:"issue"`
}

func (c *GetIncident) Name() string {
	return "jira.getIncident"
}

func (c *GetIncident) Label() string {
	return "Get Incident"
}

func (c *GetIncident) Description() string {
	return "Fetch a Jira Service Management incident by issue id or issue key"
}

func (c *GetIncident) Documentation() string {
	return `The Get Incident component returns incident details from Jira Service Management.

## Use Cases

- **Enrichment**: Load incident DTO after an issue webhook or workflow step
- **Status checks**: Read priority, status, responders, and affected services from JSM

## Configuration

- **Project** (optional): When set, the issue picker lists issues in that project (by project key), paged from Jira search (up to 500, most recently updated first). Leave empty to list issues updated in roughly the last 90 days across projects you can access (same cap and ordering).
- **Issue**: Choose an issue by key (from Jira search). The stored value is the issue key; numeric ids still work when they are saved as the issue reference.

## Output

Payload type jira.incident.fetched with the incident object returned by Atlassian (summary, reporter, priority, status, responders, etc.).`
}

func (c *GetIncident) Icon() string {
	return "jira"
}

func (c *GetIncident) Color() string {
	return "blue"
}

func (c *GetIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIncident) Configuration() []configuration.Field {
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
			Description: "Issue key (or id) of the incident; keys are resolved via the Jira REST API",
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

func (c *GetIncident) Setup(ctx core.SetupContext) error {
	spec := GetIncidentSpec{}
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

func (c *GetIncident) Execute(ctx core.ExecutionContext) error {
	spec := GetIncidentSpec{}
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

	incident, err := client.GetIncident(cloudID, issueID)
	if err != nil {
		return fmt.Errorf("failed to get incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetJiraIncidentPayloadType,
		[]any{incident},
	)
}

func (c *GetIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
