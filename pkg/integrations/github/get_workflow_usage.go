package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflowUsage struct{}

type GetWorkflowUsageConfiguration struct {
	Repositories []string `mapstructure:"repositories,omitempty" json:"repositories,omitempty"`
	Year         *string  `mapstructure:"year,omitempty" json:"year,omitempty"`
	Month        *string  `mapstructure:"month,omitempty" json:"month,omitempty"`
	Day          *string  `mapstructure:"day,omitempty" json:"day,omitempty"`
	Product      *string  `mapstructure:"product,omitempty" json:"product,omitempty"`
	SKU          *string  `mapstructure:"sku,omitempty" json:"sku,omitempty"`
}

type billingUsageSummaryResponse struct {
	UsageItems []billingUsageItem `json:"usageItems"`
}

type billingUsageItem struct {
	Date        *string  `json:"date,omitempty"`
	Product     *string  `json:"product,omitempty"`
	SKU         *string  `json:"sku,omitempty"`
	UnitType    *string  `json:"unitType,omitempty"`
	Quantity    *float64 `json:"quantity,omitempty"`
	NetQuantity *float64 `json:"netQuantity,omitempty"`
	GrossAmount *float64 `json:"grossAmount,omitempty"`
	NetAmount   *float64 `json:"netAmount,omitempty"`
}

func (c *GetWorkflowUsage) Name() string {
	return "github.getWorkflowUsage"
}

func (c *GetWorkflowUsage) Label() string {
	return "Get Workflow Usage"
}

func (c *GetWorkflowUsage) Description() string {
	return "Retrieve billable GitHub Actions usage (minutes) for an organization"
}

func (c *GetWorkflowUsage) Documentation() string {
	return `The Get Workflow Usage component retrieves **billable** GitHub Actions usage for an organization using GitHub's billing usage API.

## Important: Authentication

GitHub's organization billing endpoints are **not accessible** using GitHub App installation tokens. If you see a 403 with ` + "`Resource not accessible by integration`" + `, you must provide a **user access token**.

To enable this component, set a GitHub integration secret named ` + "`accessToken`" + ` containing a GitHub **PAT** (or OAuth token) that has permission to read the organization's billing/usage.

## Notes

- Only **private repositories** on **GitHub-hosted runners** accrue billable minutes. Public repos and self-hosted runners generally show **0 billable usage**.
- Breakdown is available **by runner OS (SKU)** (and optionally by repository if you select repositories). Breakdown **by workflow** is not available via the GitHub API.

## Configuration

- **Repositories** (optional): If empty, returns organization-wide usage. If one or more repositories are selected, usage is scoped and aggregated across those repositories.
- **Time range** (optional): ` + "`year`, `month`, `day`" + ` for billing period. Defaults to current month when omitted.
- **Product** (optional): Defaults to ` + "`Actions`" + `.
- **Runner OS / SKU** (optional): Filter to a specific SKU (e.g. ` + "`actions_linux`" + `, ` + "`actions_windows`" + `).

## Output

Emits a single object containing:
- ` + "`minutes_used`" + `: total billable minutes
- ` + "`minutes_used_breakdown`" + `: map of SKU -> minutes
- optionally ` + "`net_amount`" + ` (if provided by the API)`
}

func (c *GetWorkflowUsage) ExampleOutput() map[string]any {
	return map[string]any{
		"minutes_used": 123.0,
		"minutes_used_breakdown": map[string]any{
			"actions_linux":   100.0,
			"actions_windows": 23.0,
		},
		"product": "Actions",
		"year":    2026,
		"month":   2,
	}
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
			Description: "Optional repository scope. If empty, returns organization-wide usage.",
		},
		{
			Name:        "year",
			Label:       "Year",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., 2026",
			Description: "Optional billing year.",
		},
		{
			Name:        "month",
			Label:       "Month",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., 2",
			Description: "Optional billing month (1-12).",
		},
		{
			Name:        "day",
			Label:       "Day",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., 8",
			Description: "Optional billing day (1-31).",
		},
		{
			Name:        "product",
			Label:       "Product",
			Type:        configuration.FieldTypeString,
			Default:     "Actions",
			Description: "Billing product to query (default: Actions).",
		},
		{
			Name:        "sku",
			Label:       "Runner OS / SKU",
			Type:        configuration.FieldTypeString,
			Placeholder: "e.g., actions_linux",
			Description: "Optional SKU filter (e.g., actions_linux, actions_windows, actions_macos).",
		},
	}
}

func (c *GetWorkflowUsage) Setup(ctx core.SetupContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Repositories) == 0 {
		return nil
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	for _, repo := range config.Repositories {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}

		found := false
		for _, r := range appMetadata.Repositories {
			if r.Name == repo {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("repository %s is not accessible to app installation", repo)
		}
	}

	// Store the first selected repository in node metadata for UX consistency.
	first := strings.TrimSpace(config.Repositories[0])
	if first == "" {
		return nil
	}
	for i, r := range appMetadata.Repositories {
		if r.Name == first {
			return ctx.Metadata.Set(NodeMetadata{Repository: &appMetadata.Repositories[i]})
		}
	}

	return nil
}

func (c *GetWorkflowUsage) Execute(ctx core.ExecutionContext) error {
	var config GetWorkflowUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	year, month, day, err := parseOptionalYMD(config.Year, config.Month, config.Day)
	if err != nil {
		return err
	}

	product := "Actions"
	if config.Product != nil && strings.TrimSpace(*config.Product) != "" {
		product = strings.TrimSpace(*config.Product)
	}
	sku := ""
	if config.SKU != nil && strings.TrimSpace(*config.SKU) != "" {
		sku = strings.TrimSpace(*config.SKU)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if strings.TrimSpace(appMetadata.Owner) == "" {
		return errors.New("organization is required (GitHub integration must be installed to an organization)")
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	var tokenClient *github.Client
	if token, ok, err := findSecretOptional(ctx.Integration, GitHubAccessToken); err != nil {
		return fmt.Errorf("failed to read optional GitHub access token secret: %w", err)
	} else if ok {
		tokenClient = NewTokenClient(token)
	}

	repos := make([]string, 0, len(config.Repositories))
	for _, r := range config.Repositories {
		r = strings.TrimSpace(r)
		if r != "" {
			repos = append(repos, r)
		}
	}

	totalMinutes := 0.0
	totalNetAmount := 0.0
	minutesBySKU := map[string]float64{}

	consume := func(summary *billingUsageSummaryResponse) {
		for _, item := range summary.UsageItems {
			if item.Product == nil || strings.TrimSpace(*item.Product) == "" {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(*item.Product), product) {
				continue
			}

			if item.UnitType == nil || !strings.EqualFold(strings.TrimSpace(*item.UnitType), "minutes") {
				continue
			}

			itemSKU := ""
			if item.SKU != nil {
				itemSKU = strings.TrimSpace(*item.SKU)
			}
			if sku != "" && !strings.EqualFold(itemSKU, sku) {
				continue
			}

			minutes := 0.0
			if item.NetQuantity != nil {
				minutes = *item.NetQuantity
			} else if item.Quantity != nil {
				minutes = *item.Quantity
			}

			netAmount := 0.0
			if item.NetAmount != nil {
				netAmount = *item.NetAmount
			}

			totalMinutes += minutes
			totalNetAmount += netAmount
			if itemSKU != "" {
				minutesBySKU[itemSKU] += minutes
			}
		}
	}

	if len(repos) == 0 {
		summary, err := fetchBillingUsageSummaryWithFallback(client, tokenClient, appMetadata.Owner, "", year, month, day, product, sku)
		if err != nil {
			return wrapBillingUsageSummaryError(err, tokenClient != nil)
		}
		consume(summary)
	} else {
		for _, repo := range repos {
			repoFull := fmt.Sprintf("%s/%s", strings.TrimSpace(appMetadata.Owner), repo)
			summary, err := fetchBillingUsageSummaryWithFallback(client, tokenClient, appMetadata.Owner, repoFull, year, month, day, product, sku)
			if err != nil {
				return wrapBillingUsageSummaryError(err, tokenClient != nil)
			}
			consume(summary)
		}
	}

	out := map[string]any{
		"minutes_used":           totalMinutes,
		"minutes_used_breakdown": minutesBySKU,
		"product":                product,
	}
	if sku != "" {
		out["sku"] = sku
	}
	if len(repos) > 0 {
		out["repositories"] = repos
	}
	if year != nil {
		out["year"] = *year
	}
	if month != nil {
		out["month"] = *month
	}
	if day != nil {
		out["day"] = *day
	}
	if totalNetAmount != 0 {
		out["net_amount"] = totalNetAmount
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.workflowUsage",
		[]any{out},
	)
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

func parseOptionalYMD(year, month, day *string) (*int, *int, *int, error) {
	parseOpt := func(name string, v *string) (*int, error) {
		if v == nil || strings.TrimSpace(*v) == "" {
			return nil, nil
		}
		i, err := strconv.Atoi(strings.TrimSpace(*v))
		if err != nil {
			return nil, fmt.Errorf("%s is not a number: %v", name, err)
		}
		return &i, nil
	}

	y, err := parseOpt("year", year)
	if err != nil {
		return nil, nil, nil, err
	}
	m, err := parseOpt("month", month)
	if err != nil {
		return nil, nil, nil, err
	}
	d, err := parseOpt("day", day)
	if err != nil {
		return nil, nil, nil, err
	}

	if y != nil && *y <= 0 {
		return nil, nil, nil, errors.New("year must be greater than 0")
	}
	if m != nil && (*m < 1 || *m > 12) {
		return nil, nil, nil, errors.New("month must be between 1 and 12")
	}
	if d != nil && (*d < 1 || *d > 31) {
		return nil, nil, nil, errors.New("day must be between 1 and 31")
	}

	// Enforce hierarchy: can't specify day without month, or month/day without year.
	if d != nil && m == nil {
		return nil, nil, nil, errors.New("month is required when day is set")
	}
	if (m != nil || d != nil) && y == nil {
		return nil, nil, nil, errors.New("year is required when month/day is set")
	}

	return y, m, d, nil
}

func fetchBillingUsageSummaryWithFallback(
	appClient *github.Client,
	tokenClient *github.Client,
	organization string,
	repository string,
	year, month, day *int,
	product string,
	sku string,
) (*billingUsageSummaryResponse, error) {
	summary, err := fetchBillingUsageSummary(appClient, organization, repository, year, month, day, product, sku)
	if err == nil {
		return summary, nil
	}

	// Billing endpoints can return 403 "Resource not accessible by integration" for
	// GitHub App installation tokens. Retry with a user token if configured.
	if tokenClient != nil && isResourceNotAccessibleByIntegration(err) {
		return fetchBillingUsageSummary(tokenClient, organization, repository, year, month, day, product, sku)
	}

	return nil, err
}

func fetchBillingUsageSummary(
	client *github.Client,
	organization string,
	repository string,
	year, month, day *int,
	product string,
	sku string,
) (*billingUsageSummaryResponse, error) {
	values := url.Values{}
	if year != nil {
		values.Set("year", strconv.Itoa(*year))
	}
	if month != nil {
		values.Set("month", strconv.Itoa(*month))
	}
	if day != nil {
		values.Set("day", strconv.Itoa(*day))
	}
	if strings.TrimSpace(repository) != "" {
		values.Set("repository", strings.TrimSpace(repository))
	}
	if strings.TrimSpace(product) != "" {
		values.Set("product", strings.TrimSpace(product))
	}
	if strings.TrimSpace(sku) != "" {
		values.Set("sku", strings.TrimSpace(sku))
	}

	path := fmt.Sprintf("organizations/%s/settings/billing/usage/summary", organization)
	if q := values.Encode(); q != "" {
		path = path + "?" + q
	}

	req, err := client.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	var out billingUsageSummaryResponse
	_, err = client.Do(context.Background(), req, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func isForbidden(err error) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == 403 {
		return true
	}
	return false
}

func isResourceNotAccessibleByIntegration(err error) bool {
	var ghErr *github.ErrorResponse
	if !errors.As(err, &ghErr) || ghErr == nil || ghErr.Response == nil {
		return false
	}
	if ghErr.Response.StatusCode != 403 {
		return false
	}
	return strings.Contains(strings.ToLower(ghErr.Message), "resource not accessible by integration")
}

func wrapBillingUsageSummaryError(err error, hasToken bool) error {
	if isResourceNotAccessibleByIntegration(err) {
		if !hasToken {
			return fmt.Errorf(
				"GitHub billing usage endpoints are not accessible via GitHub App installation tokens. Configure a GitHub integration secret named %q with a PAT/OAuth token that can read org billing usage: %w",
				GitHubAccessToken,
				err,
			)
		}

		return fmt.Errorf("permission denied (403): GitHub token does not have access to organization billing usage: %w", err)
	}

	if isForbidden(err) {
		return fmt.Errorf("permission denied (403): %w", err)
	}

	return err
}
