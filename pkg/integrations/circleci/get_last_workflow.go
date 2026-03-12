package circleci

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetLastWorkflowPayloadType = "circleci.workflow"
const maxGetLastWorkflowPipelinePages = 20

type GetLastWorkflow struct{}

type GetLastWorkflowConfiguration struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	Branch      string `json:"branch" mapstructure:"branch"`
	Status      string `json:"status" mapstructure:"status"`
}

func (c *GetLastWorkflow) Name() string {
	return "circleci.getLastWorkflow"
}

func (c *GetLastWorkflow) Label() string {
	return "Get Last Workflow"
}

func (c *GetLastWorkflow) Description() string {
	return "Get the most recent workflow for a CircleCI project"
}

func (c *GetLastWorkflow) Documentation() string {
	return `The Get Last Workflow component retrieves the most recent workflow for a CircleCI project.

## Use Cases

- **Latest status check**: Get the most recent workflow to check project health
- **Branch monitoring**: Monitor the latest workflow on a specific branch
- **Status filtering**: Find the last workflow with a specific status (e.g., last successful build)

## How It Works

1. Fetches recent pipelines for the project (optionally filtered by branch)
2. Iterates through pipelines to find workflows
3. Returns the first workflow matching the optional status filter

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/username/repo)
- **Branch**: Optional branch filter
- **Status**: Optional workflow status filter (success, failed, etc.)

## Output

Emits a ` + "`circleci.workflow`" + ` payload with the most recent matching workflow details.`
}

func (c *GetLastWorkflow) Icon() string {
	return "workflow"
}

func (c *GetLastWorkflow) Color() string {
	return "gray"
}

func (c *GetLastWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetLastWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "projectSlug",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Optional branch to filter pipelines",
		},
		{
			Name:        "status",
			Label:       "Status filter",
			Type:        configuration.FieldTypeSelect,
			Description: "Optional: only return a workflow with this status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Any", Value: "any"},
						{Label: "Success", Value: "success"},
						{Label: "Failed", Value: "failed"},
						{Label: "Running", Value: "running"},
						{Label: "On Hold", Value: "on_hold"},
						{Label: "Canceled", Value: "canceled"},
						{Label: "Error", Value: "error"},
					},
				},
			},
		},
	}
}

func decodeGetLastWorkflowConfiguration(config any) (GetLastWorkflowConfiguration, error) {
	spec := GetLastWorkflowConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return GetLastWorkflowConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ProjectSlug = strings.TrimSpace(spec.ProjectSlug)
	if spec.ProjectSlug == "" {
		return GetLastWorkflowConfiguration{}, fmt.Errorf("project slug is required")
	}

	return spec, nil
}

func (c *GetLastWorkflow) Setup(ctx core.SetupContext) error {
	_, err := decodeGetLastWorkflowConfiguration(ctx.Configuration)
	return err
}

func (c *GetLastWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetLastWorkflow) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetLastWorkflowConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	nextPageToken := ""
	seenPageTokens := map[string]struct{}{}
	pageCount := 0
	for {
		pageCount++
		if pageCount > maxGetLastWorkflowPipelinePages {
			return fmt.Errorf("failed to get last workflow: exceeded maximum pipeline pages (%d)", maxGetLastWorkflowPipelinePages)
		}

		pipelines, err := client.GetProjectPipelinesWithPageToken(spec.ProjectSlug, spec.Branch, nextPageToken)
		if err != nil {
			return fmt.Errorf("failed to get project pipelines: %w", err)
		}

		for _, pipeline := range pipelines.Items {
			workflows, err := client.GetPipelineWorkflows(pipeline.ID)
			if err != nil {
				return fmt.Errorf("failed to get workflows for pipeline %s: %w", pipeline.ID, err)
			}

			for _, workflow := range workflows {
				if spec.Status != "" && spec.Status != "any" && workflow.Status != spec.Status {
					continue
				}

				payload := map[string]any{
					"id":         workflow.ID,
					"name":       workflow.Name,
					"status":     workflow.Status,
					"createdAt":  workflow.CreatedAt,
					"stoppedAt":  workflow.StoppedAt,
					"pipelineId": pipeline.ID,
				}

				return ctx.ExecutionState.Emit(
					core.DefaultOutputChannel.Name,
					GetLastWorkflowPayloadType,
					[]any{payload},
				)
			}
		}

		if pipelines.NextPageToken == "" {
			break
		}

		if _, exists := seenPageTokens[pipelines.NextPageToken]; exists {
			return fmt.Errorf("failed to get last workflow: detected pagination cycle for pipeline pages")
		}

		seenPageTokens[pipelines.NextPageToken] = struct{}{}
		nextPageToken = pipelines.NextPageToken
	}

	return fmt.Errorf("no matching workflow found")
}

func (c *GetLastWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetLastWorkflow) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetLastWorkflow) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetLastWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetLastWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}
