package github

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflowUsage struct{}

type GetWorkflowUsageConfiguration struct {
	// Note: Repository is optional for this component since billing is at org level
}

// GetWorkflowUsageOutput represents the output of the GetWorkflowUsage component
type GetWorkflowUsageOutput struct {
	TotalMinutesUsed     float64            `json:"total_minutes_used"`
	TotalPaidMinutesUsed float64            `json:"total_paid_minutes_used"`
	IncludedMinutes      float64            `json:"included_minutes"`
	MinutesUsedBreakdown map[string]int     `json:"minutes_used_breakdown"`
	Organization         string             `json:"organization"`
}

func (c *GetWorkflowUsage) Name() string {
	return "github.getWorkflowUsage"
}

func (c *GetWorkflowUsage) Label() string {
	return "Get Workflow Usage"
}

func (c *GetWorkflowUsage) Description() string {
	return "Retrieve GitHub Actions usage (billable minutes) for the organization"
}

func (c *GetWorkflowUsage) Documentation() string {
	return `The Get Workflow Usage component retrieves billable GitHub Actions usage (minutes) for the organization associated with the integration's GitHub App installation.

## Use Cases

- **Billing monitoring**: Check Actions usage for billing or quota awareness from SuperPlane workflows
- **Cost reporting**: Report on workflow run minutes (e.g., monthly) for cost or compliance purposes
- **Usage alerts**: Compare usage against thresholds to alert when approaching limits
- **Resource planning**: Track usage patterns to plan for capacity and costs

## Configuration

This component requires no additional configuration. It automatically uses the organization from the GitHub App installation.

## Output

Returns usage data including:
- **total_minutes_used**: Total billable minutes consumed in the current billing cycle
- **total_paid_minutes_used**: Minutes beyond the included free tier
- **included_minutes**: Free minutes included in the plan
- **minutes_used_breakdown**: Breakdown by runner OS (Linux, Windows, macOS)
- **organization**: The organization name

## Notes

- Only private repositories on GitHub-hosted runners accrue billable minutes
- Public repositories and self-hosted runners show zero billable usage
- The GitHub App requires **Organization Administration (read)** permission to access billing data
- Existing app installations may need to approve the new permission before this component works

## Permissions

This component requires the GitHub App to have **Administration (read)** organization permission. If this permission was not granted when the app was installed, organization owners will be prompted by GitHub to approve the new permission; until they do, this component will return a 403 error for those installations.`
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
	// No additional configuration needed - uses org from integration metadata
	return []configuration.Field{}
}

func (c *GetWorkflowUsage) Setup(ctx core.SetupContext) error {
	// Validate that we have the necessary integration metadata
	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if appMetadata.Owner == "" {
		return fmt.Errorf("organization/owner is not set in integration metadata")
	}

	if appMetadata.InstallationID == "" {
		return fmt.Errorf("installation ID is not set in integration metadata")
	}

	return nil
}

func (c *GetWorkflowUsage) Execute(ctx core.ExecutionContext) error {
	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Get Actions billing for the organization
	billing, _, err := client.Billing.GetActionsBillingOrg(
		context.Background(),
		appMetadata.Owner,
	)
	if err != nil {
		return fmt.Errorf("failed to get Actions billing for organization %s: %w", appMetadata.Owner, err)
	}

	// Build output
	output := GetWorkflowUsageOutput{
		TotalMinutesUsed:     billing.TotalMinutesUsed,
		TotalPaidMinutesUsed: billing.TotalPaidMinutesUsed,
		IncludedMinutes:      billing.IncludedMinutes,
		MinutesUsedBreakdown: billing.MinutesUsedBreakdown,
		Organization:         appMetadata.Owner,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.workflowUsage",
		[]any{output},
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
