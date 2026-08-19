package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_pipeline_minutes_usage.json
var exampleOutputGetPipelineMinutesUsage []byte

type GetPipelineMinutesUsage struct{}

type GetPipelineMinutesUsageConfiguration struct {
	Projects []string `mapstructure:"projects"`
}

type PipelineMinutesUsageResult struct {
	Month                 string                  `json:"month"`
	MonthIso8601          string                  `json:"monthIso8601"`
	Minutes               int                     `json:"minutes"`
	SharedRunnersDuration int                     `json:"sharedRunnersDuration"`
	Projects              []CiMinutesProjectUsage `json:"projects"`
}

func (c *GetPipelineMinutesUsage) Name() string {
	return "gitlab.getPipelineMinutesUsage"
}

func (c *GetPipelineMinutesUsage) Label() string {
	return "Get Pipeline Minutes Usage"
}

func (c *GetPipelineMinutesUsage) Description() string {
	return "Retrieve CI/CD compute (pipeline) minutes usage for the connected GitLab namespace"
}

func (c *GetPipelineMinutesUsage) Documentation() string {
	return `The Get Pipeline Minutes Usage component retrieves CI/CD compute minutes usage for the current calendar month, for the namespace configured on the connected GitLab integration (its group, or the personal namespace if no group is set).

## Prerequisites

This action calls GitLab's GraphQL API, which requires the connected user to have access to the namespace's usage quota data (at least the Owner role for groups).

## Behavior

- Returns usage for the current calendar month
- Includes a per-project breakdown of minutes and shared runner duration
- When Projects are selected, only usage for those projects is included in the totals
- If the integration's Group is a subgroup, usage is reported for its root group, since GitLab only tracks compute minutes at the root namespace

## Configuration

- **Projects** (optional, multiselect): Limit the per-project breakdown and totals to specific projects. Leave empty to include all projects with usage.

## Output

Returns:
- ` + "`month`" + `: The usage month name, e.g. "July"
- ` + "`monthIso8601`" + `: The usage month as an ISO date, e.g. "2026-07-01"
- ` + "`minutes`" + `: Total compute minutes used
- ` + "`sharedRunnersDuration`" + `: Total shared runner duration, in seconds
- ` + "`projects`" + `: Per-project breakdown (minutes, shared runner duration, and project details)

## Use Cases

- **Billing monitoring**: Track CI/CD minutes usage against the namespace's quota
- **Cost control**: Alert when usage approaches a budget threshold
- **Usage reporting**: Generate periodic usage reports

## References

- [GitLab compute minutes](https://docs.gitlab.com/ci/pipelines/compute_minutes/)`
}

func (c *GetPipelineMinutesUsage) Icon() string {
	return "gitlab"
}

func (c *GetPipelineMinutesUsage) Color() string {
	return "orange"
}

func (c *GetPipelineMinutesUsage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPipelineMinutesUsage) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetPipelineMinutesUsage, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetPipelineMinutesUsage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projects",
			Label:       "Projects",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Limit the per-project breakdown and totals to specific projects. Leave empty to include all projects with usage.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeProject,
					Multi: true,
				},
			},
		},
	}
}

func (c *GetPipelineMinutesUsage) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetPipelineMinutesUsage) Execute(ctx core.ExecutionContext) error {
	var config GetPipelineMinutesUsageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	namespaceGID, err := resolveNamespaceGID(client)
	if err != nil {
		return err
	}

	date := time.Now().Format("2006-01") + "-01"
	usage, projects, err := client.GetCiMinutesUsage(context.Background(), namespaceGID, date)
	if err != nil {
		return fmt.Errorf("failed to get pipeline minutes usage: %w", err)
	}

	if len(config.Projects) > 0 {
		projects = filterProjectsUsage(projects, config.Projects)
	}

	result := PipelineMinutesUsageResult{Projects: projects}
	if usage != nil {
		result.Month = usage.Month
		result.MonthIso8601 = usage.MonthIso8601
		result.Minutes = usage.Minutes
		result.SharedRunnersDuration = usage.SharedRunnersDuration
	}

	// Recompute totals from the per-project breakdown whenever the namespace record is missing or a project filter narrows it, so totals never silently disagree with what's shown.
	if usage == nil || len(config.Projects) > 0 {
		result.Minutes, result.SharedRunnersDuration = sumProjectUsage(projects)
	}

	if usage == nil {
		result.MonthIso8601 = date
		if parsed, err := time.Parse("2006-01-02", date); err == nil {
			result.Month = parsed.Format("January")
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.pipelineMinutesUsage",
		[]any{result},
	)
}

// resolveNamespaceGID turns the connected integration's configured group into a GraphQL namespace ID, or nil to use the current user's personal namespace.
func resolveNamespaceGID(client *Client) (*string, error) {
	if client.groupID == "" {
		return nil, nil
	}

	group, err := client.GetGroup(client.groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve group: %w", err)
	}

	// GitLab tracks compute minutes on the root namespace only, so a subgroup's own ID would resolve to a namespace with no usage records.
	rootPath, _, isSubgroup := strings.Cut(group.FullPath, "/")
	if isSubgroup {
		group, err = client.GetGroup(rootPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve root group: %w", err)
		}
	}

	gid := fmt.Sprintf("gid://gitlab/Group/%d", group.ID)
	return &gid, nil
}

// sumProjectUsage totals minutes and shared runner duration across a per-project usage breakdown.
func sumProjectUsage(projects []CiMinutesProjectUsage) (minutes, sharedRunnersDuration int) {
	for _, p := range projects {
		minutes += p.Minutes
		sharedRunnersDuration += p.SharedRunnersDuration
	}
	return minutes, sharedRunnersDuration
}

// filterProjectsUsage keeps only the usage entries for projects whose numeric ID is in selected.
func filterProjectsUsage(projects []CiMinutesProjectUsage, selected []string) []CiMinutesProjectUsage {
	filtered := make([]CiMinutesProjectUsage, 0, len(projects))
	for _, p := range projects {
		if p.Project == nil {
			continue
		}
		if slices.Contains(selected, globalIDSuffix(p.Project.ID)) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// globalIDSuffix extracts the trailing numeric ID from a GitLab GraphQL Global ID, e.g. "gid://gitlab/Project/123" -> "123".
func globalIDSuffix(gid string) string {
	idx := strings.LastIndex(gid, "/")
	if idx == -1 {
		return gid
	}
	return gid[idx+1:]
}

func (c *GetPipelineMinutesUsage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetPipelineMinutesUsage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPipelineMinutesUsage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetPipelineMinutesUsage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetPipelineMinutesUsage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
