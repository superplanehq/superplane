package github

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetBillingUsage struct{}

type GetBillingUsageConfiguration struct {
	Repositories []string `json:"repositories" mapstructure:"repositories"`
	Year         string   `json:"year" mapstructure:"year"`
	Month        string   `json:"month" mapstructure:"month"`
	Day          string   `json:"day" mapstructure:"day"`
	Product      string   `json:"product" mapstructure:"product"`
	SKU          string   `json:"sku" mapstructure:"sku"`
}

type BillingUsageOutput struct {
	MinutesUsed          int64             `json:"minutes_used"`
	MinutesUsedBreakdown map[string]int64  `json:"minutes_used_breakdown"`
	TotalCost            float64           `json:"total_cost,omitempty"`
	RepositoryBreakdown  []RepositoryUsage `json:"repository_breakdown,omitempty"`
}

type RepositoryUsage struct {
	RepositoryName string           `json:"repository_name"`
	MinutesUsed    int64            `json:"minutes_used"`
	Breakdown      map[string]int64 `json:"breakdown"`
}

func (c *GetBillingUsage) Name() string {
	return "github.getBillingUsage"
}

func (c *GetBillingUsage) Label() string {
	return "Get Billing Usage"
}

func (c *GetBillingUsage) Description() string {
	return "Retrieve billable GitHub Actions usage (minutes) for the organization"
}

func (c *GetBillingUsage) Documentation() string {
	return `The Get Billing Usage component retrieves billable GitHub Actions usage (minutes) for the installation's organization.

## Prerequisites

This action requires the GitHub App to have **Organization permission: Administration (read)**. Existing installations must approve this new permission when prompted by GitHub. Until approved, this action will return a 403 error.

**Note**: This component uses GitHub's enhanced billing usage report API, which provides detailed usage information and is the recommended approach for accessing billing data.

## Use Cases

- **Billing monitoring**: Check Actions usage for billing or quota from SuperPlane workflows
- **Monthly reporting**: Generate reports on workflow run minutes for cost tracking or compliance
- **Usage alerts**: Compare usage to thresholds and trigger alerts when approaching limits
- **Cost allocation**: Track usage per repository for internal chargeback

## How It Works

This component calls GitHub's usage report API to retrieve:
- Total billable minutes for the specified time period
- Breakdown by runner OS (Linux, Windows, macOS)
- Optional per-repository breakdown when specific repositories are selected
- Optional cost information (if available from the API)

**Note**: Only private repositories on GitHub-hosted runners accrue billable minutes. Public repos and self-hosted runners show zero billable usage.

## Configuration

- **Repositories** (optional, multi-select): List of repositories to include. When empty, returns usage for all repos in the organization. When one or more selected, scopes to those repos only.
- **Year** (optional): Year for billing period (e.g., "2026"). Defaults to current year.
- **Month** (optional): Month for billing period (1-12). Defaults to current month.
- **Day** (optional): Day for billing period (1-31). When specified, narrows to a specific day.
- **Product** (optional): Billing product to query. Defaults to "actions". Can be used for other products like "copilot" in the future.
- **Runner OS / SKU** (optional): Filter to a specific runner OS (e.g., "UBUNTU", "WINDOWS", "MACOS"). When unset, returns usage for all runner OSes.

## Output

Returns usage data on the default output channel:
- **minutes_used**: Total billable minutes in the period
- **minutes_used_breakdown**: Minutes broken down by OS (Linux/Windows/macOS)
- **total_cost**: Optional cost information (if available from API)
- **repository_breakdown**: Per-repository usage when multiple repos are selected

**Important**: Breakdown is by OS (runner SKU) and repository, not by individual workflow. Workflow-level breakdown is not available via the GitHub API.

## Error Handling

Errors do not emit a payload. Common errors include:
- **403 Forbidden**: Administration permission not granted on the installation
- **404 Not Found**: Repository or organization not found
- **5xx errors**: GitHub API issues`
}

func (c *GetBillingUsage) Icon() string {
	return "chart-bar"
}

func (c *GetBillingUsage) Color() string {
	return "gray"
}

func (c *GetBillingUsage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetBillingUsage) Configuration() []configuration.Field {
	currentTime := time.Now()
	return []configuration.Field{
		{
			Name:     "repositories",
			Label:    "Repositories",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
			Description: "Leave empty to include all repositories in the organization. Select specific repositories to scope usage data.",
		},
		{
			Name:        "year",
			Label:       "Year",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     fmt.Sprintf("%d", currentTime.Year()),
			Description: "Year for billing period (e.g., 2026). Defaults to current year.",
		},
		{
			Name:        "month",
			Label:       "Month",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     fmt.Sprintf("%d", int(currentTime.Month())),
			Description: "Month for billing period (1-12). Defaults to current month.",
		},
		{
			Name:        "day",
			Label:       "Day",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Day for billing period (1-31). Leave empty for monthly summary.",
		},
		{
			Name:        "product",
			Label:       "Product",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "actions",
			Description: "Billing product to query. Defaults to 'actions'.",
		},
		{
			Name:        "sku",
			Label:       "Runner OS / SKU",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter to specific runner OS (e.g., UBUNTU, WINDOWS, MACOS). Leave empty for all.",
		},
	}
}

func (c *GetBillingUsage) Setup(ctx core.SetupContext) error {
	var config GetBillingUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate year/month if provided
	if config.Year != "" {
		year, err := strconv.Atoi(config.Year)
		if err != nil || year < 2000 || year > 3000 {
			return fmt.Errorf("invalid year: %s", config.Year)
		}
	}

	if config.Month != "" {
		month, err := strconv.Atoi(config.Month)
		if err != nil || month < 1 || month > 12 {
			return fmt.Errorf("invalid month: %s (must be 1-12)", config.Month)
		}
	}

	if config.Day != "" {
		day, err := strconv.Atoi(config.Day)
		if err != nil || day < 1 || day > 31 {
			return fmt.Errorf("invalid day: %s (must be 1-31)", config.Day)
		}
	}

	// Ensure selected repositories are in metadata
	if len(config.Repositories) > 0 {
		for _, repo := range config.Repositories {
			err := ensureRepoInMetadata(
				ctx.Metadata,
				ctx.Integration,
				map[string]any{"repository": repo},
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *GetBillingUsage) Execute(ctx core.ExecutionContext) error {
	var config GetBillingUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Set defaults for time period
	if config.Year == "" {
		config.Year = fmt.Sprintf("%d", time.Now().Year())
	}
	if config.Month == "" {
		config.Month = fmt.Sprintf("%d", int(time.Now().Month()))
	}
	if config.Product == "" {
		config.Product = "actions"
	}

	var output BillingUsageOutput
	output.MinutesUsedBreakdown = make(map[string]int64)

	// Get usage data for organization
	usage, err := c.getDetailedUsage(context.Background(), client, appMetadata.Owner, config)
	if err != nil {
		return fmt.Errorf("failed to get usage data: %w", err)
	}
	output = usage

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.billing.usage",
		[]any{output},
	)
}

func (c *GetBillingUsage) getDetailedUsage(goCtx context.Context, client *github.Client, owner string, config GetBillingUsageConfiguration) (BillingUsageOutput, error) {
	// Build options for the usage report API
	opts := &github.UsageReportOptions{}

	// Note: Year, Month, and Day are already validated in Setup(), so errors can be safely ignored here
	if config.Year != "" {
		year, _ := strconv.Atoi(config.Year)
		opts.Year = &year
	}
	if config.Month != "" {
		month, _ := strconv.Atoi(config.Month)
		opts.Month = &month
	}
	if config.Day != "" {
		day, _ := strconv.Atoi(config.Day)
		opts.Day = &day
	}

	// Get usage report
	report, _, err := client.Billing.GetUsageReportOrg(goCtx, owner, opts)
	if err != nil {
		return BillingUsageOutput{}, fmt.Errorf("failed to get usage report: %w", err)
	}

	output := BillingUsageOutput{
		MinutesUsedBreakdown: make(map[string]int64),
	}

	// Track repositories if needed
	repoUsageMap := make(map[string]*RepositoryUsage)

	// Process usage items
	for _, item := range report.UsageItems {
		// Filter by product if not actions
		if config.Product != "" && config.Product != "actions" && item.GetProduct() != config.Product {
			continue
		}

		// Only process Actions-related items
		if item.GetProduct() != "actions" {
			continue
		}

		// Filter by SKU if specified
		if config.SKU != "" && item.GetSKU() != config.SKU {
			continue
		}

		// Filter by repository if specified
		repoName := item.GetRepositoryName()
		if len(config.Repositories) > 0 {
			found := false
			for _, repo := range config.Repositories {
				if repoName == repo {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		quantity := item.GetQuantity()
		if quantity == nil {
			continue
		}
		quantityInt := int64(*quantity)
		sku := item.GetSKU()

		// Add to totals
		output.MinutesUsed += quantityInt
		output.MinutesUsedBreakdown[sku] += quantityInt

		// Add cost if available
		if item.NetAmount != nil {
			output.TotalCost += *item.NetAmount
		}

		// Track per-repository breakdown if we have multiple repos or filtering by repos
		if len(config.Repositories) > 0 && repoName != "" {
			if repoUsageMap[repoName] == nil {
				repoUsageMap[repoName] = &RepositoryUsage{
					RepositoryName: repoName,
					Breakdown:      make(map[string]int64),
				}
			}
			repoUsageMap[repoName].MinutesUsed += quantityInt
			repoUsageMap[repoName].Breakdown[sku] += quantityInt
		}
	}

	// Convert repo map to slice
	if len(repoUsageMap) > 0 {
		output.RepositoryBreakdown = []RepositoryUsage{}
		for _, repoUsage := range repoUsageMap {
			output.RepositoryBreakdown = append(output.RepositoryBreakdown, *repoUsage)
		}
	}

	return output, nil
}

func (c *GetBillingUsage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetBillingUsage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetBillingUsage) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetBillingUsage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetBillingUsage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetBillingUsage) Cleanup(ctx core.SetupContext) error {
	return nil
}
