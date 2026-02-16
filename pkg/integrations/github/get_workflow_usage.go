package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflowUsage struct{}

type GetWorkflowUsageConfiguration struct {
	Repositories []string `mapstructure:"repositories"`
	Year         *string  `mapstructure:"year,omitempty"`
	Month        *string  `mapstructure:"month,omitempty"`
	Day          *string  `mapstructure:"day,omitempty"`
	Product      string   `mapstructure:"product"`
	SKU          *string  `mapstructure:"sku,omitempty"`
}

type githubUsageItem struct {
	Product        string   `json:"product"`
	SKU            string   `json:"sku"`
	Quantity       *float64 `json:"quantity"`
	UnitType       string   `json:"unitType"`
	NetAmount      *float64 `json:"netAmount"`
	RepositoryName string   `json:"repositoryName"`
}

type githubBillingUsageResponse struct {
	UsageItems []githubUsageItem `json:"usageItems"`
}

type githubWorkflowUsageOutput struct {
	MinutesUsed          float64            `json:"minutes_used"`
	MinutesUsedBreakdown map[string]float64 `json:"minutes_used_breakdown"`
	NetAmount            float64            `json:"net_amount"`
	UsageItems           []githubUsageItem  `json:"usage_items"`
	Repositories         []string           `json:"repositories,omitempty"`
}

func (c *GetWorkflowUsage) Name() string {
	return "github.getWorkflowUsage"
}

func (c *GetWorkflowUsage) Label() string {
	return "Get Workflow Usage"
}

func (c *GetWorkflowUsage) Description() string {
	return "Get billable GitHub Actions workflow usage for an organization"
}

func (c *GetWorkflowUsage) Documentation() string {
	return `The Get Workflow Usage component retrieves billable GitHub Actions usage for the installation organization.

## Prerequisites

- GitHub App must have **Organization Administration: Read** permission.
- Existing app installations may need approval for the new permission before this action can access billing usage.

## Use Cases

- **Billing visibility**: Report monthly billable workflow minutes
- **Cost monitoring**: Track usage by runner OS (Linux / Windows / macOS)
- **Repository filtering**: Restrict usage to selected repositories

## Configuration

- **Repositories**: Optional multi-select of repositories. Empty means all repositories in the organization.
- **Year / Month / Day**: Optional billing period filters.
- **Product**: Product to query, defaults to Actions.
- **Runner OS / SKU**: Optional OS/SKU filter.

## Output

Returns a single payload containing:
- **minutes_used**: Total billable minutes
- **minutes_used_breakdown**: Minutes by OS/SKU
- **net_amount**: Aggregated net cost (when returned by API)
- **usage_items**: Filtered raw usage rows

Note: GitHub billing usage provides breakdown by SKU/OS and repository, not by workflow.`
}

func (c *GetWorkflowUsage) Icon() string {
	return "github"
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
			Name:  "repositories",
			Label: "Repositories",
			Type:  configuration.FieldTypeIntegrationResource,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
			Description: "Optional repository filter. Leave empty to query usage for all repositories in the organization.",
		},
		{
			Name:        "year",
			Label:       "Year",
			Type:        configuration.FieldTypeString,
			Description: "Optional year (for example: 2026).",
		},
		{
			Name:        "month",
			Label:       "Month",
			Type:        configuration.FieldTypeString,
			Description: "Optional month 1-12.",
		},
		{
			Name:        "day",
			Label:       "Day",
			Type:        configuration.FieldTypeString,
			Description: "Optional day 1-31.",
		},
		{
			Name:     "product",
			Label:    "Product",
			Type:     configuration.FieldTypeSelect,
			Default:  "Actions",
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Actions",
							Value: "Actions",
						},
						{
							Label: "Copilot",
							Value: "Copilot",
						},
						{
							Label: "Packages",
							Value: "Packages",
						},
					},
				},
			},
			Description: "Billing product to query.",
		},
		{
			Name:    "sku",
			Label:   "Runner OS / SKU",
			Type:    configuration.FieldTypeSelect,
			Default: "",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "All",
							Value: "",
						},
						{
							Label: "Linux (actions_linux)",
							Value: "actions_linux",
						},
						{
							Label: "Windows (actions_windows)",
							Value: "actions_windows",
						},
						{
							Label: "macOS (actions_macos)",
							Value: "actions_macos",
						},
						{
							Label: "macOS Larger Runner (actions_macos_larger)",
							Value: "actions_macos_larger",
						},
					},
				},
			},
			Description: "Optional SKU filter for runner OS.",
		},
	}
}

func (c *GetWorkflowUsage) Setup(ctx core.SetupContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	if len(config.Repositories) == 0 {
		return nil
	}

	known := map[string]struct{}{}
	for _, repo := range appMetadata.Repositories {
		known[repo.Name] = struct{}{}
	}

	for _, repository := range config.Repositories {
		if _, ok := known[repository]; !ok {
			return fmt.Errorf("repository %s is not accessible to app installation", repository)
		}
	}

	return nil
}

func (c *GetWorkflowUsage) Execute(ctx core.ExecutionContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	if appMetadata.Owner == "" {
		return fmt.Errorf("organization owner is missing from integration metadata")
	}

	query := url.Values{}
	if config.Year != nil && *config.Year != "" {
		year, err := strconv.Atoi(*config.Year)
		if err != nil || year < 1970 {
			return fmt.Errorf("year must be a valid number")
		}
		query.Set("year", strconv.Itoa(year))
	}
	if config.Month != nil && *config.Month != "" {
		month, err := strconv.Atoi(*config.Month)
		if err != nil || month < 1 || month > 12 {
			return fmt.Errorf("month must be between 1 and 12")
		}
		query.Set("month", strconv.Itoa(month))
	}
	if config.Day != nil && *config.Day != "" {
		day, err := strconv.Atoi(*config.Day)
		if err != nil || day < 1 || day > 31 {
			return fmt.Errorf("day must be between 1 and 31")
		}
		query.Set("day", strconv.Itoa(day))
	}

	product := strings.TrimSpace(config.Product)
	if product == "" {
		product = "Actions"
	}
	query.Set("product", product)

	if config.SKU != nil && strings.TrimSpace(*config.SKU) != "" {
		query.Set("sku", strings.TrimSpace(*config.SKU))
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	path := fmt.Sprintf("orgs/%s/settings/billing/usage", appMetadata.Owner)
	if query.Encode() != "" {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	req, err := client.NewRequest("GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create usage request: %w", err)
	}

	var response githubBillingUsageResponse
	if _, err = client.Do(context.Background(), req, &response); err != nil {
		return fmt.Errorf("failed to fetch billing usage: %w", err)
	}

	selectedRepos := map[string]struct{}{}
	for _, repository := range config.Repositories {
		selectedRepos[repository] = struct{}{}
	}

	filtered := make([]githubUsageItem, 0, len(response.UsageItems))
	minutes := 0.0
	netAmount := 0.0
	breakdown := map[string]float64{}

	for _, item := range response.UsageItems {
		if len(selectedRepos) > 0 {
			if _, ok := selectedRepos[item.RepositoryName]; !ok {
				continue
			}
		}

		filtered = append(filtered, item)

		if item.NetAmount != nil {
			netAmount += *item.NetAmount
		}

		if item.Quantity == nil {
			continue
		}

		unit := strings.ToLower(item.UnitType)
		if !strings.Contains(unit, "minute") {
			continue
		}

		minutes += *item.Quantity
		osKey := normalizeUsageSKU(item.SKU)
		breakdown[osKey] += *item.Quantity
	}

	output := githubWorkflowUsageOutput{
		MinutesUsed:          minutes,
		MinutesUsedBreakdown: breakdown,
		NetAmount:            netAmount,
		UsageItems:           filtered,
		Repositories:         config.Repositories,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.workflowUsage",
		[]any{output},
	)
}

func normalizeUsageSKU(sku string) string {
	value := strings.ToLower(strings.TrimSpace(sku))
	switch value {
	case "actions_linux":
		return "linux"
	case "actions_windows":
		return "windows"
	case "actions_macos", "actions_macos_larger":
		return "macos"
	default:
		if value == "" {
			return "unknown"
		}
		return value
	}
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
