package github

import (
	"context"
	"fmt"
	"slices"

	gh "github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflowUsage struct{}

type GetWorkflowUsageConfiguration struct {
	Repositories []string `mapstructure:"repositories"`
}

type GetWorkflowUsageMetadata struct {
	Repositories []RepositoryMetadata `json:"repositories" mapstructure:"repositories"`
}

type RepositoryMetadata struct {
	ID   int64  `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
	URL  string `json:"url" mapstructure:"url"`
}

type WorkflowUsageResult struct {
	MinutesUsed          float64                 `json:"minutes_used" mapstructure:"minutes_used"`
	MinutesUsedBreakdown gh.MinutesUsedBreakdown `json:"minutes_used_breakdown" mapstructure:"minutes_used_breakdown"`
	IncludedMinutes      float64                 `json:"included_minutes" mapstructure:"included_minutes"`
	TotalPaidMinutesUsed float64                 `json:"total_paid_minutes_used" mapstructure:"total_paid_minutes_used"`
	Repositories         []string                `json:"repositories,omitempty" mapstructure:"repositories,omitempty"`
}

func (g *GetWorkflowUsage) Name() string {
	return "github.getWorkflowUsage"
}

func (g *GetWorkflowUsage) Label() string {
	return "Get Workflow Usage"
}

func (g *GetWorkflowUsage) Description() string {
	return "Retrieve billable GitHub Actions usage (minutes) for the organization"
}

func (g *GetWorkflowUsage) Documentation() string {
	return `The Get Workflow Usage component retrieves billable GitHub Actions usage (minutes) for the installation's organization.

## Prerequisites

This action calls GitHub's **billing usage** API, which requires the GitHub App to have **Organization permission: Organization administration (read)**. 

**Important**: Existing installations will need to approve the new permission when prompted by GitHub. Until the permission is granted, this action will return a 403 error.

**Note**: This component uses GitHub's enhanced billing usage report API, which provides detailed usage information.

## Behavior

- Returns billing data for the **current billing cycle** 
- Only private repositories on GitHub-hosted runners accrue billable minutes
- Public repositories and self-hosted runners show zero billable usage
- Can filter by specific repositories when selected
- Uses enhanced billing platform API for accurate reporting

## Configuration

- **Repositories** (optional, multiselect): Select one or more specific repositories to track. These will be included in the output for reference (max 5) and stored in node metadata with full repository details (ID, name, URL). When repositories are selected, only usage for those repositories is included in the totals.

## Output

Returns usage data with:
- ` + "`minutes_used`" + `: Total billable minutes used in the current billing cycle
- ` + "`minutes_used_breakdown`" + `: Map of minutes by runner SKU (e.g., "Actions Linux": 120, "Actions Windows": 60, "Actions macOS": 30)
- ` + "`included_minutes`" + `: Always 0 (not provided by enhanced billing API)
- ` + "`total_paid_minutes_used`" + `: Estimated paid minutes based on cost data
- ` + "`repositories`" + `: List of selected repositories for tracking (max 5)

**Note**: Breakdown is by runner SKU (OS and type), not by individual workflow.

## Node Metadata

The component stores repository information in node metadata:
- Repository ID, name, and URL for each selected repository (max 5)
- This metadata is displayed in the workflow canvas for easy reference

## Use Cases

- **Billing Monitoring**: Track GitHub Actions usage for billing purposes
- **Quota Management**: Monitor usage to avoid exceeding billing quotas
- **Cost Control**: Alert when usage approaches limits or budget thresholds
- **Usage Reporting**: Generate monthly or periodic usage reports for compliance
- **Resource Planning**: Analyze runner usage patterns by OS type

## References

- [GitHub Billing Usage API](https://docs.github.com/rest/billing/usage)
- [GitHub Enhanced Billing Platform](https://docs.github.com/billing/using-the-new-billing-platform)
- [Permissions required for GitHub Apps - Organization Administration](https://docs.github.com/en/rest/overview/permissions-required-for-github-apps#organization-permissions-for-administration)
- [Viewing your usage of metered products](https://docs.github.com/en/billing/managing-billing-for-github-actions/viewing-your-github-actions-usage)`
}

func (g *GetWorkflowUsage) Icon() string {
	return "github"
}

func (g *GetWorkflowUsage) Color() string {
	return "gray"
}

func (g *GetWorkflowUsage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetWorkflowUsage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repositories",
			Label:       "Repositories",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Select specific repositories to check usage for. Leave empty for organization-wide usage.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
	}
}

func (g *GetWorkflowUsage) Setup(ctx core.SetupContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Repositories) > 0 {
		selectedRepos, err := validateAndCollectRepositories(ctx, config.Repositories)
		if err != nil {
			return err
		}

		reposToStore := selectedRepos
		if len(reposToStore) > 5 {
			reposToStore = reposToStore[:5]
		}

		metadata := GetWorkflowUsageMetadata{
			Repositories: reposToStore,
		}

		return ctx.Metadata.Set(metadata)
	}

	return nil
}

func (g *GetWorkflowUsage) Execute(ctx core.ExecutionContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	report, _, err := client.Billing.GetUsageReportOrg(
		context.Background(),
		appMetadata.Owner,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to get billing usage: %w", err)
	}

	result := aggregateUsageData(report, config.Repositories)

	if len(config.Repositories) > 0 {
		reposToInclude := config.Repositories
		if len(reposToInclude) > 5 {
			reposToInclude = reposToInclude[:5]
		}
		result.Repositories = reposToInclude
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.workflowUsage",
		[]any{result},
	)
}

func (g *GetWorkflowUsage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetWorkflowUsage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (g *GetWorkflowUsage) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetWorkflowUsage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetWorkflowUsage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetWorkflowUsage) Cleanup(ctx core.SetupContext) error {
	return nil
}

// validateAndCollectRepositories validates that the specified repositories exist in the app metadata
// and collects their full repository objects.
func validateAndCollectRepositories(ctx core.SetupContext, repoNames []string) ([]RepositoryMetadata, error) {
	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode application metadata: %w", err)
	}

	var selectedRepos []RepositoryMetadata
	for _, repoName := range repoNames {
		repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
			return r.Name == repoName
		})

		if repoIndex == -1 {
			return nil, fmt.Errorf("repository %s is not accessible to app installation", repoName)
		}

		availableRepo := appMetadata.Repositories[repoIndex]
		selectedRepos = append(selectedRepos, RepositoryMetadata{
			ID:   availableRepo.ID,
			Name: availableRepo.Name,
			URL:  availableRepo.URL,
		})
	}

	return selectedRepos, nil
}

// aggregateUsageData processes the billing usage report and aggregates usage data.
// If repositories are specified, only usage for those repositories is included.
func aggregateUsageData(report *gh.UsageReport, repositories []string) WorkflowUsageResult {
	result := WorkflowUsageResult{
		MinutesUsed:          0,
		MinutesUsedBreakdown: make(gh.MinutesUsedBreakdown),
		IncludedMinutes:      0,
		TotalPaidMinutesUsed: 0,
	}

	for _, item := range report.UsageItems {
		if item.GetProduct() != "actions" {
			continue
		}

		if len(repositories) > 0 {
			repoName := item.GetRepositoryName()
			if !slices.Contains(repositories, repoName) {
				continue
			}
		}

		if item.Quantity != nil {
			result.MinutesUsed += *item.Quantity
		}

		sku := item.GetSKU()
		if sku != "" && item.Quantity != nil {
			result.MinutesUsedBreakdown[sku] = result.MinutesUsedBreakdown[sku] + int(*item.Quantity)
		}

		if item.NetAmount != nil && *item.NetAmount > 0 {
			result.TotalPaidMinutesUsed += *item.NetAmount / 0.008
		}
	}

	return result
}
