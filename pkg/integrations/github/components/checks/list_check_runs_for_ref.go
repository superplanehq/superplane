package checks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type ListCheckRunsForRef struct{}

type ListCheckRunsForRefConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Ref        string `json:"ref" mapstructure:"ref"`
	CheckName  string `json:"checkName" mapstructure:"checkName"`
	Status     string `json:"status" mapstructure:"status"`
	Filter     string `json:"filter" mapstructure:"filter"`
}

func (c *ListCheckRunsForRef) Name() string {
	return "github.listCheckRunsForRef"
}

func (c *ListCheckRunsForRef) Label() string {
	return "List Check Runs For Ref"
}

func (c *ListCheckRunsForRef) Description() string {
	return "List GitHub Checks API check runs for a commit, branch, or tag"
}

func (c *ListCheckRunsForRef) Documentation() string {
	return `The List Check Runs For Ref component retrieves GitHub Checks API check runs for a commit SHA, branch, or tag.

Use it after On Check Run or other GitHub triggers to inspect the full set of Checks API results for the same ref before deciding whether to continue.

## Use Cases

- **PR quality gates**: Continue only when all check runs for a PR commit are complete and green
- **Check aggregation**: Inspect the full Checks API state after one check run changes
- **Notifications**: Build summaries for failed or pending check runs

## Configuration

- **Repository**: Select the GitHub repository
- **Ref**: Commit SHA, branch, or tag ref
- **Check Name** *(optional)*: Return only check runs with this name
- **Status** *(optional)*: Return only check runs with this status
- **Filter**: Use latest to return the latest run per check suite, or all to include all matching runs

## Output

Returns GitHub's check run list response, including total_count and check_runs. Each check run includes its name, status, conclusion, URLs, app metadata, associated pull requests, and timestamps.`
}

func (c *ListCheckRunsForRef) Icon() string {
	return "github"
}

func (c *ListCheckRunsForRef) Color() string {
	return "gray"
}

func (c *ListCheckRunsForRef) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListCheckRunsForRef) Configuration() []configuration.Field {
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
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "Commit SHA, branch, or tag",
			Description: "Commit SHA, branch name, heads/<branch>, tags/<tag>, or tag name.",
		},
		{
			Name:        "checkName",
			Label:       "Check Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. DCO",
			Description: "Optional. Return check runs with this exact name.",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional. Return check runs with this status.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Queued", Value: "queued"},
						{Label: "In Progress", Value: "in_progress"},
						{Label: "Completed", Value: "completed"},
					},
				},
			},
		},
		{
			Name:     "filter",
			Label:    "Filter",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "latest",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Latest", Value: "latest"},
						{Label: "All", Value: "all"},
					},
				},
			},
		},
	}
}

func (c *ListCheckRunsForRef) Setup(ctx core.SetupContext) error {
	return common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration)
}

func (c *ListCheckRunsForRef) Execute(ctx core.ExecutionContext) error {
	var config ListCheckRunsForRefConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	result, err := listAllCheckRunsForRef(client, config)
	if err != nil {
		return fmt.Errorf("failed to list check runs for ref: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "github.checkRuns", []any{result})
}

func (c *ListCheckRunsForRef) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListCheckRunsForRef) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ListCheckRunsForRef) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListCheckRunsForRef) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ListCheckRunsForRef) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ListCheckRunsForRef) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func buildListCheckRunsOptions(config ListCheckRunsForRefConfiguration) *github.ListCheckRunsOptions {
	options := &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	if config.CheckName != "" {
		options.CheckName = github.Ptr(config.CheckName)
	}

	if config.Status != "" {
		options.Status = github.Ptr(config.Status)
	}

	if config.Filter != "" {
		options.Filter = github.Ptr(config.Filter)
	}

	return options
}

func listAllCheckRunsForRef(client *common.Client, config ListCheckRunsForRefConfiguration) (*github.ListCheckRunsResults, error) {
	options := buildListCheckRunsOptions(config)
	allResults := &github.ListCheckRunsResults{}

	for {
		result, response, err := client.ListCheckRunsForRef(context.Background(), config.Repository, config.Ref, options)
		if err != nil {
			return nil, err
		}

		if result != nil {
			allResults.CheckRuns = append(allResults.CheckRuns, result.CheckRuns...)
			allResults.Total = result.Total
		}

		if response == nil || response.NextPage == 0 {
			break
		}

		options.Page = response.NextPage
	}

	return allResults, nil
}
