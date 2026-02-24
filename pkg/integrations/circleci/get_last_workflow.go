package circleci

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetLastWorkflow struct{}

type GetLastWorkflowConfiguration struct {
	ProjectSlug string `json:"projectSlug" mapstructure:"projectSlug"`
	Branch      string `json:"branch" mapstructure:"branch"`
	Status      string `json:"status" mapstructure:"status"`
}

type GetLastWorkflowResult struct {
	ProjectSlug string            `json:"project_slug"`
	Pipeline    *PipelineResponse `json:"pipeline"`
	Workflow    *WorkflowResponse `json:"workflow"`
}

func (c *GetLastWorkflow) Name() string {
	return "circleci.getLastWorkflow"
}

func (c *GetLastWorkflow) Label() string {
	return "Get Last Workflow"
}

func (c *GetLastWorkflow) Description() string {
	return "Get the most recent workflow for a project, with optional branch and status filters"
}

func (c *GetLastWorkflow) Documentation() string {
	return `The Get Last Workflow component retrieves the latest workflow run in a CircleCI project.

## Use Cases

- **Latest run checks**: Fetch the newest workflow status before continuing a workflow
- **Branch-specific monitoring**: Inspect only workflows from a specific branch
- **Status filtering**: Find the most recent workflow matching a status such as success or failed

## Configuration

- **Project Slug**: CircleCI project slug (for example ` + "`gh/org/repo`" + `)
- **Branch** (optional): Filter pipelines to a specific branch
- **Status** (optional): Return the latest workflow whose status matches this value (case-insensitive)

## Output

Returns:
- ` + "`project_slug`" + `: Project slug used in the request
- ` + "`pipeline`" + `: Pipeline details for the matched workflow
- ` + "`workflow`" + `: The matched workflow`
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
			Name:        "projectSlug",
			Label:       "Project Slug",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI project slug (e.g. gh/org/repo)",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional branch filter",
			Placeholder: "e.g. main",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional workflow status filter (e.g. success, failed, running)",
			Placeholder: "e.g. success",
		},
	}
}

func (c *GetLastWorkflow) Setup(ctx core.SetupContext) error {
	var config GetLastWorkflowConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.ProjectSlug) == "" {
		return fmt.Errorf("projectSlug is required")
	}

	return nil
}

func (c *GetLastWorkflow) Execute(ctx core.ExecutionContext) error {
	var config GetLastWorkflowConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	projectSlug := strings.TrimSpace(config.ProjectSlug)
	branch := strings.TrimSpace(config.Branch)
	statusFilter := strings.TrimSpace(config.Status)
	if projectSlug == "" {
		return fmt.Errorf("projectSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	pipelines, err := client.GetProjectPipelines(projectSlug, branch)
	if err != nil {
		return fmt.Errorf("failed to list project pipelines: %w", err)
	}
	if len(pipelines) == 0 {
		return fmt.Errorf("no pipelines found for project %s", projectSlug)
	}

	sortPipelinesByRecency(pipelines)

	for _, pipeline := range pipelines {
		workflows, err := client.GetPipelineWorkflows(pipeline.ID)
		if err != nil {
			return fmt.Errorf("failed to list workflows for pipeline %s: %w", pipeline.ID, err)
		}

		sortWorkflowsByRecency(workflows)

		for _, workflow := range workflows {
			if statusFilter != "" && !strings.EqualFold(workflow.Status, statusFilter) {
				continue
			}

			selectedPipeline := pipeline
			selectedWorkflow := workflow
			result := GetLastWorkflowResult{
				ProjectSlug: projectSlug,
				Pipeline:    &selectedPipeline,
				Workflow:    &selectedWorkflow,
			}

			return ctx.ExecutionState.Emit(
				core.DefaultOutputChannel.Name,
				"circleci.lastWorkflow",
				[]any{result},
			)
		}
	}

	if statusFilter != "" {
		return fmt.Errorf("no workflow found for project %s with status %s", projectSlug, statusFilter)
	}

	return fmt.Errorf("no workflows found for project %s", projectSlug)
}

func (c *GetLastWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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

func sortPipelinesByRecency(pipelines []PipelineResponse) {
	sort.SliceStable(pipelines, func(i, j int) bool {
		iTime, iOK := parseRFC3339(pipelines[i].CreatedAt)
		jTime, jOK := parseRFC3339(pipelines[j].CreatedAt)

		if iOK && jOK {
			return iTime.After(jTime)
		}
		if iOK {
			return true
		}
		if jOK {
			return false
		}

		return pipelines[i].Number > pipelines[j].Number
	})
}

func sortWorkflowsByRecency(workflows []WorkflowResponse) {
	sort.SliceStable(workflows, func(i, j int) bool {
		iTime, iOK := parseRFC3339(workflows[i].CreatedAt)
		jTime, jOK := parseRFC3339(workflows[j].CreatedAt)

		if iOK && jOK {
			return iTime.After(jTime)
		}
		if iOK {
			return true
		}
		if jOK {
			return false
		}

		return workflows[i].PipelineNumber > workflows[j].PipelineNumber
	})
}

func parseRFC3339(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}
