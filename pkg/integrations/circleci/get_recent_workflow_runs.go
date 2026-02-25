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

const GetRecentWorkflowRunsPayloadType = "circleci.workflowRuns"

type GetRecentWorkflowRuns struct{}

type GetRecentWorkflowRunsConfiguration struct {
	ProjectSlug  string `json:"projectSlug" mapstructure:"projectSlug"`
	WorkflowName string `json:"workflowName" mapstructure:"workflowName"`
	Branch       string `json:"branch" mapstructure:"branch"`
}

func (c *GetRecentWorkflowRuns) Name() string {
	return "circleci.getRecentWorkflowRuns"
}

func (c *GetRecentWorkflowRuns) Label() string {
	return "Get Recent Workflow Runs"
}

func (c *GetRecentWorkflowRuns) Description() string {
	return "Get recent runs of a CircleCI workflow"
}

func (c *GetRecentWorkflowRuns) Documentation() string {
	return `The Get Recent Workflow Runs component fetches recent individual runs for a named CircleCI workflow via the Insights API.

## Use Cases

- **Workflow health monitoring**: See recent run statuses and durations at a glance
- **Performance tracking**: Monitor how long workflow runs take over time
- **Branch comparison**: Compare recent runs across branches

## How It Works

1. Calls the CircleCI Insights endpoint for the given project and workflow name
2. Returns a list of individual workflow runs (up to 90 days back)
3. Each run includes its status, duration, branch, timestamps, and credits used

## Configuration

- **Project Slug**: CircleCI project slug (e.g., gh/username/repo)
- **Workflow Name**: Name of the workflow to fetch runs for
- **Branch**: Optional branch filter (defaults to the project's default branch)

## Output

Emits a ` + "`circleci.workflowRuns`" + ` payload containing an array of recent workflow runs with fields like ` + "`id`" + `, ` + "`status`" + `, ` + "`duration`" + `, ` + "`branch`" + `, ` + "`createdAt`" + `, ` + "`stoppedAt`" + `, and ` + "`creditsUsed`" + `.`
}

func (c *GetRecentWorkflowRuns) Icon() string {
	return "workflow"
}

func (c *GetRecentWorkflowRuns) Color() string {
	return "gray"
}

func (c *GetRecentWorkflowRuns) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRecentWorkflowRuns) Configuration() []configuration.Field {
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
			Name:     "workflowName",
			Label:    "Workflow",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeWorkflow,
					Parameters: []configuration.ParameterRef{
						{
							Name: "projectSlug",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "projectSlug",
							},
						},
					},
				},
			},
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Description: "Optional branch filter (defaults to project default branch)",
		},
	}
}

func decodeGetRecentWorkflowRunsConfiguration(config any) (GetRecentWorkflowRunsConfiguration, error) {
	spec := GetRecentWorkflowRunsConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return GetRecentWorkflowRunsConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.ProjectSlug = strings.TrimSpace(spec.ProjectSlug)
	if spec.ProjectSlug == "" {
		return GetRecentWorkflowRunsConfiguration{}, fmt.Errorf("project slug is required")
	}

	spec.WorkflowName = strings.TrimSpace(spec.WorkflowName)
	if spec.WorkflowName == "" {
		return GetRecentWorkflowRunsConfiguration{}, fmt.Errorf("workflow name is required")
	}

	return spec, nil
}

func (c *GetRecentWorkflowRuns) Setup(ctx core.SetupContext) error {
	_, err := decodeGetRecentWorkflowRunsConfiguration(ctx.Configuration)
	return err
}

func (c *GetRecentWorkflowRuns) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRecentWorkflowRuns) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetRecentWorkflowRunsConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	runs, err := client.GetWorkflowRuns(spec.ProjectSlug, spec.WorkflowName, WorkflowRunsParams{
		Branch: spec.Branch,
	})
	if err != nil {
		return fmt.Errorf("failed to get recent workflow runs: %w", err)
	}

	runList := make([]map[string]any, 0, len(runs.Items))
	for _, r := range runs.Items {
		runList = append(runList, map[string]any{
			"id":          r.ID,
			"branch":      r.Branch,
			"duration":    r.Duration,
			"createdAt":   r.CreatedAt,
			"stoppedAt":   r.StoppedAt,
			"creditsUsed": r.CreditsUsed,
			"status":      r.Status,
			"isApproval":  r.IsApproval,
		})
	}

	payload := map[string]any{
		"runs": runList,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetRecentWorkflowRunsPayloadType,
		[]any{payload},
	)
}

func (c *GetRecentWorkflowRuns) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetRecentWorkflowRuns) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRecentWorkflowRuns) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRecentWorkflowRuns) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRecentWorkflowRuns) Cleanup(ctx core.SetupContext) error {
	return nil
}
