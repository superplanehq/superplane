package github

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflowUsage struct{}

type GetWorkflowUsageConfiguration struct {
	Repositories []string `json:"repositories" mapstructure:"repositories"`
	Year         string   `json:"year" mapstructure:"year"`
	Month        string   `json:"month" mapstructure:"month"`
	Day          string   `json:"day" mapstructure:"day"`
	Product      string   `json:"product" mapstructure:"product"`
	SKU          string   `json:"sku" mapstructure:"sku"`
}

type WorkflowUsageOutput struct {
	MinutesUsed          int                       `json:"minutes_used"`
	MinutesUsedBreakdown map[string]int            `json:"minutes_used_breakdown"`
	TotalPaidMinutes     int                       `json:"total_paid_minutes_used"`
	IncludedMinutes      int                       `json:"included_minutes"`
	NetAmount            float64                   `json:"net_amount,omitempty"`
	Repositories         []RepositoryUsageBreakdown `json:"repositories,omitempty"`
}

type RepositoryUsageBreakdown struct {
	Name        string `json:"name"`
	MinutesUsed int    `json:"minutes_used"`
}

func (c *GetWorkflowUsage) Name() string {
	return "github.getWorkflowUsage"
}

func (c *GetWorkflowUsage) Label() string {
	return "Get Workflow Usage"
}

func (c *GetWorkflowUsage) Description() string {
	return "Retrieve billable GitHub Actions usage for selected repositories"
}

func (c *GetWorkflowUsage) Documentation() string {
	return `The Get Workflow Usage component retrieves billable GitHub Actions minutes for the organization, optionally scoped to specific repositories and time periods.

## Use Cases

- **Billing monitoring**: Track Actions usage for cost management
- **Quota alerts**: Check usage against limits and trigger alerts
- **Cost reporting**: Generate monthly or weekly usage reports
- **Budget tracking**: Monitor workflow run costs across teams

## Configuration

- **Repositories**: Select repositories to include. Leave empty for org-wide usage.
- **Year**: Filter by year (optional, defaults to current year)
- **Month**: Filter by month (optional, defaults to current month)
- **Day**: Filter by day (optional)
- **Product**: Billing product (default: Actions)
- **Runner OS / SKU**: Filter by runner type (e.g., actions_linux, actions_windows)

## Output

Returns usage data including:
- Total billable minutes used
- Breakdown by OS/runner type
- Per-repository breakdown (when multiple repos selected)
- Cost information (if available)

## Note

Only private repositories on GitHub-hosted runners accrue billable minutes. Public repositories and self-hosted runners show zero billable usage.`
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
			Name:     "repositories",
			Label:    "Repositories",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:   "repository",
					Search: true,
					Multi:  true,
				},
			},
			Description: "Select repositories to include. Leave empty for organization-wide usage.",
		},
		{
			Name:        "year",
			Label:       "Year",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by year (YYYY). Defaults to current year.",
		},
		{
			Name:        "month",
			Label:       "Month",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by month (1-12). Defaults to current month.",
		},
		{
			Name:        "day",
			Label:       "Day",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by day (1-31). Optional.",
		},
		{
			Name:        "product",
			Label:       "Product",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Billing product (default: Actions).",
		},
		{
			Name:        "sku",
			Label:       "Runner OS / SKU",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by runner type (e.g., actions_linux, actions_windows, actions_macos).",
		},
	}
}

func (c *GetWorkflowUsage) Execute(ctx core.ExecutionContext) ([]core.OutputChannel, error) {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration(), &config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	client, err := getClient(ctx.SyncContext())
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	owner := ctx.SyncContext().Integration.Metadata["owner"].(string)

	// Set defaults for time params
	now := time.Now()
	year := config.Year
	if year == "" {
		year = strconv.Itoa(now.Year())
	}
	month := config.Month
	if month == "" {
		month = strconv.Itoa(int(now.Month()))
	}

	product := config.Product
	if product == "" {
		product = "Actions"
	}

	// Build query params
	params := fmt.Sprintf("?year=%s&month=%s&product=%s", year, month, product)
	if config.Day != "" {
		params += fmt.Sprintf("&day=%s", config.Day)
	}
	if config.SKU != "" {
		params += fmt.Sprintf("&sku=%s", config.SKU)
	}

	var usageData WorkflowUsageOutput
	breakdown := make(map[string]int)
	var repoBreakdowns []RepositoryUsageBreakdown

	if len(config.Repositories) == 0 {
		// Org-wide usage
		url := fmt.Sprintf("/orgs/%s/settings/billing/actions%s", owner, params)
		req, err := client.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		var response struct {
			TotalMinutesUsed     int     `json:"total_minutes_used"`
			TotalPaidMinutesUsed int     `json:"total_paid_minutes_used"`
			IncludedMinutes      int     `json:"included_minutes"`
			MinutesUsedBreakdown map[string]int `json:"minutes_used_breakdown"`
		}

		if _, err := client.Do(context.Background(), req, &response); err != nil {
			return nil, fmt.Errorf("failed to fetch org usage: %w", err)
		}

		usageData.MinutesUsed = response.TotalPaidMinutesUsed
		usageData.TotalPaidMinutes = response.TotalPaidMinutesUsed
		usageData.IncludedMinutes = response.IncludedMinutes
		usageData.MinutesUsedBreakdown = response.MinutesUsedBreakdown
	} else {
		// Per-repo usage
		totalMinutes := 0
		for _, repoID := range config.Repositories {
			repoName := repoID
			url := fmt.Sprintf("/repos/%s/%s/actions/billing/usage%s", owner, repoName, params)
			req, err := client.NewRequest("GET", url, nil)
			if err != nil {
				continue
			}

			var response struct {
				TotalMinutesUsed     int     `json:"total_minutes_used"`
				TotalPaidMinutesUsed int     `json:"total_paid_minutes_used"`
				IncludedMinutes      int     `json:"included_minutes"`
				MinutesUsedBreakdown map[string]int `json:"minutes_used_breakdown"`
			}

			if _, err := client.Do(context.Background(), req, &response); err != nil {
				continue
			}

			totalMinutes += response.TotalPaidMinutesUsed
			repoBreakdowns = append(repoBreakdowns, RepositoryUsageBreakdown{
				Name:        repoName,
				MinutesUsed: response.TotalPaidMinutesUsed,
			})

			// Aggregate OS breakdown
			for os, minutes := range response.MinutesUsedBreakdown {
				breakdown[os] += minutes
			}
		}

		usageData.MinutesUsed = totalMinutes
		usageData.TotalPaidMinutes = totalMinutes
		usageData.MinutesUsedBreakdown = breakdown
		usageData.Repositories = repoBreakdowns
	}

	return []core.OutputChannel{
		{
			Name: core.DefaultOutputChannel.Name,
			Output: map[string]any{
				"minutes_used":           usageData.MinutesUsed,
				"total_paid_minutes":     usageData.TotalPaidMinutes,
				"included_minutes":       usageData.IncludedMinutes,
				"minutes_used_breakdown": usageData.MinutesUsedBreakdown,
				"repositories":           usageData.Repositories,
			},
		},
	}, nil
}
