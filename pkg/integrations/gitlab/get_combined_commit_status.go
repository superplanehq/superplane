package gitlab

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_get_combined_commit_status.json
var exampleOutputGetCombinedCommitStatus []byte

const CombinedCommitStatusPayloadType = "gitlab.combinedCommitStatus"

// combinedStatePriority orders GitLab build states worst-first; the combined
// state is the worst state present among a commit's statuses.
var combinedStatePriority = []string{
	"failed", "canceled", "running", "pending", "manual", "created", "success", "skipped",
}

type GetCombinedCommitStatus struct{}

type GetCombinedCommitStatusConfiguration struct {
	Project string `mapstructure:"project"`
	Ref     string `mapstructure:"ref"`
}

func (c *GetCombinedCommitStatus) Name() string {
	return "gitlab.getCombinedCommitStatus"
}

func (c *GetCombinedCommitStatus) Label() string {
	return "Get Combined Commit Status"
}

func (c *GetCombinedCommitStatus) Description() string {
	return "Get the combined commit status for a GitLab commit, branch, or tag"
}

func (c *GetCombinedCommitStatus) Documentation() string {
	return `The Get Combined Commit Status component reads the combined status for a commit, branch, or tag - a single overall state rolled up from the commit's individual statuses.

GitLab has no dedicated combined-status endpoint, so this component lists the commit's statuses and rolls them into one overall **state** (the worst state present), alongside the full **statuses** list and a **total_count**.

## Use Cases

- **Status gates**: Check whether a commit is green before continuing
- **Follow-up to Publish Commit Status**: Read back the aggregate status after reporting one
- **Notifications**: Send a concise summary when a commit is blocked by a failed status

## Configuration

- **Project** (required): The GitLab project containing the commit
- **Ref** (required): Commit SHA, branch name, or tag name to inspect (supports expressions)

## Permissions

The connected token needs the ` + "`read_api`" + ` (or ` + "`api`" + `) scope and at least the **Reporter** role on the project.

## Output

Emits a combined commit status object: the overall **state**, the **sha**, the **total_count**, and the latest status per context in **statuses**.`
}

func (c *GetCombinedCommitStatus) Icon() string {
	return "gitlab"
}

func (c *GetCombinedCommitStatus) Color() string {
	return "orange"
}

func (c *GetCombinedCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCombinedCommitStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetCombinedCommitStatus, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetCombinedCommitStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
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
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "main, v1.2.3, or {{ event.data.checkout_sha }}",
			Description: "Commit SHA, branch name, or tag name to inspect. Supports expressions.",
		},
	}
}

func (c *GetCombinedCommitStatus) Setup(ctx core.SetupContext) error {
	var config GetCombinedCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if strings.TrimSpace(config.Ref) == "" {
		return errors.New("ref is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *GetCombinedCommitStatus) Execute(ctx core.ExecutionContext) error {
	var config GetCombinedCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ref := strings.TrimSpace(config.Ref)
	if ref == "" {
		return errors.New("ref is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	statuses, err := client.ListCommitStatuses(config.Project, ref, nil)
	if err != nil {
		return fmt.Errorf("failed to get combined commit status: %w", err)
	}

	if statuses == nil {
		statuses = []CommitStatus{}
	}

	sha := ref
	if len(statuses) > 0 && statuses[0].SHA != "" {
		sha = statuses[0].SHA
	}

	combined := &CombinedCommitStatus{
		State:      combinedCommitStatusState(statuses),
		SHA:        sha,
		TotalCount: len(statuses),
		Statuses:   statuses,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CombinedCommitStatusPayloadType,
		[]any{combined},
	)
}

// combinedCommitStatusState rolls a commit's statuses into one overall state
// (worst wins), mirroring GitHub's combined status. Empty when there are none.
func combinedCommitStatusState(statuses []CommitStatus) string {
	present := make(map[string]bool, len(statuses))
	for _, status := range statuses {
		present[status.Status] = true
	}

	for _, state := range combinedStatePriority {
		if present[state] {
			return state
		}
	}

	return ""
}

func (c *GetCombinedCommitStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCombinedCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetCombinedCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCombinedCommitStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCombinedCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetCombinedCommitStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
