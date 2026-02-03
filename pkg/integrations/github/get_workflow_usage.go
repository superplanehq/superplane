package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflowUsage struct{}

type GetWorkflowUsageConfiguration struct {
	Repository   string  `mapstructure:"repository"`
	WorkflowFile *string `mapstructure:"workflowFile,omitempty"`
}

func (c *GetWorkflowUsage) Name() string {
	return "github.getWorkflowUsage"
}

func (c *GetWorkflowUsage) Label() string {
	return "Get Workflow Usage"
}

func (c *GetWorkflowUsage) Description() string {
	return "Get GitHub Actions workflow usage and billing information"
}

func (c *GetWorkflowUsage) Documentation() string {
	return `The Get Workflow Usage component retrieves billable usage information for GitHub Actions workflows.

## Use Cases

- **Cost monitoring**: Track workflow execution time and billable minutes
- **Usage reporting**: Generate reports on workflow usage across runner types
- **Budget tracking**: Monitor GitHub Actions spend against budgets
- **Optimization**: Identify workflows consuming the most resources

## Configuration

- **Repository**: Select the GitHub repository
- **Workflow File**: Specific workflow file to get usage for (optional). If not specified, returns usage for all workflows.

## Output

Returns workflow usage information including:
- Total billable milliseconds per runner type (Ubuntu, macOS, Windows)
- Workflow identification (ID, name, path)

## Notes

- Usage data reflects billable minutes consumed by workflow runs
- Different runner types have different billing rates
- Self-hosted runners are not included in billable usage`
}

func (c *GetWorkflowUsage) Icon() string {
	return "workflow"
}

func (c *GetWorkflowUsage) Color() string {
	return "gray"
}

func (c *GetWorkflowUsage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetWorkflowUsage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "workflowFile",
			Label:       "Workflow File",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., ci.yml or .github/workflows/ci.yml",
			Description: "Specific workflow file to get usage for. Leave empty for all workflows.",
		},
	}
}

func (c *GetWorkflowUsage) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *GetWorkflowUsage) Execute(ctx core.ExecutionContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// If a specific workflow is requested, get usage for that workflow
	if config.WorkflowFile != nil && *config.WorkflowFile != "" {
		return c.getWorkflowUsageByFile(ctx, client, appMetadata.Owner, config.Repository, *config.WorkflowFile)
	}

	// Otherwise, get usage for all workflows
	return c.getAllWorkflowsUsage(ctx, client, appMetadata.Owner, config.Repository)
}

func (c *GetWorkflowUsage) getWorkflowUsageByFile(ctx core.ExecutionContext, client *github.Client, owner, repo, workflowFile string) error {
	// Normalize workflow file path - API expects just the filename, not full path
	normalizedFile := strings.Replace(workflowFile, ".github/workflows/", "", 1)

	// Get the workflow by filename
	workflow, _, err := client.Actions.GetWorkflowByFileName(
		context.Background(),
		owner,
		repo,
		normalizedFile,
	)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Get usage for this workflow
	usage, _, err := client.Actions.GetWorkflowUsageByID(
		context.Background(),
		owner,
		repo,
		workflow.GetID(),
	)
	if err != nil {
		return fmt.Errorf("failed to get workflow usage: %w", err)
	}

	usageData := buildWorkflowUsageData(workflow, usage)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.workflowUsage",
		[]any{usageData},
	)
}

func (c *GetWorkflowUsage) getAllWorkflowsUsage(ctx core.ExecutionContext, client *github.Client, owner, repo string) error {
	// List all workflows with pagination
	var allWorkflows []*github.Workflow
	opts := &github.ListOptions{PerPage: 100}

	for {
		workflows, resp, err := client.Actions.ListWorkflows(
			context.Background(),
			owner,
			repo,
			opts,
		)
		if err != nil {
			return fmt.Errorf("failed to list workflows: %w", err)
		}

		allWorkflows = append(allWorkflows, workflows.Workflows...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	usageList := make([]any, 0, len(allWorkflows))

	for _, workflow := range allWorkflows {
		usage, _, err := client.Actions.GetWorkflowUsageByID(
			context.Background(),
			owner,
			repo,
			workflow.GetID(),
		)
		if err != nil {
			ctx.Logger.Warnf("Failed to get usage for workflow %s: %v", workflow.GetName(), err)
			continue
		}

		usageData := buildWorkflowUsageData(workflow, usage)
		usageList = append(usageList, usageData)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.workflowUsage",
		usageList,
	)
}

func buildWorkflowUsageData(workflow *github.Workflow, usage *github.WorkflowUsage) map[string]any {
	data := map[string]any{
		"workflow": map[string]any{
			"id":         workflow.GetID(),
			"name":       workflow.GetName(),
			"path":       workflow.GetPath(),
			"state":      workflow.GetState(),
			"html_url":   workflow.GetHTMLURL(),
			"badge_url":  workflow.GetBadgeURL(),
			"created_at": workflow.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
			"updated_at": workflow.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
		},
		"billable": map[string]any{},
	}

	billable := usage.GetBillable()
	if billable != nil {
		billableData := make(map[string]any)

		// WorkflowBillMap is map[string]*WorkflowBill
		// GitHub API returns keys in UPPERCASE (UBUNTU, MACOS, WINDOWS)
		// Normalize to lowercase for frontend consistency
		for runnerType, bill := range *billable {
			if bill != nil {
				normalizedType := strings.ToLower(runnerType)
				billableData[normalizedType] = map[string]any{
					"total_ms": bill.GetTotalMS(),
				}
			}
		}

		data["billable"] = billableData
	}

	return data
}

func (c *GetWorkflowUsage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetWorkflowUsage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetWorkflowUsage) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetWorkflowUsage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetWorkflowUsage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetWorkflowUsage) Cleanup(ctx core.SetupContext) error {
	return nil
}
