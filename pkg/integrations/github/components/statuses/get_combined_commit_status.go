package statuses

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type GetCombinedCommitStatus struct{}

type GetCombinedCommitStatusConfiguration struct {
	Repository string `mapstructure:"repository"`
	Ref        string `mapstructure:"ref"`
}

func (c *GetCombinedCommitStatus) Name() string {
	return "github.getCombinedCommitStatus"
}

func (c *GetCombinedCommitStatus) Label() string {
	return "Get Combined Commit Status"
}

func (c *GetCombinedCommitStatus) Description() string {
	return "Get the combined GitHub commit status for a commit, branch, or tag"
}

func (c *GetCombinedCommitStatus) Documentation() string {
	return `The Get Combined Commit Status component reads GitHub's combined commit status for a commit, branch, or tag.

This component uses the Commit Statuses API. It summarizes legacy commit statuses, such as statuses posted by external CI systems through GitHub's statuses endpoint. It does not include GitHub Checks API check runs.

## Use Cases

- **Status gates**: Check whether all commit status contexts are green before continuing
- **Pull request automation**: Pair with On Commit Status to re-evaluate a PR when one status changes
- **Branch protection helpers**: Inspect the aggregate status for a branch head or merge commit
- **Notifications**: Send concise status summaries when a commit is blocked by failed or pending statuses

## Configuration

- **Repository**: Select the GitHub repository
- **Ref**: Commit SHA, branch name, or tag name to inspect. Expressions are supported.

## Output

Emits a combined commit status object on the default output channel. The payload includes the combined **state**, **sha**, **total_count**, and the latest status for each status context in **statuses**.`
}

func (c *GetCombinedCommitStatus) Icon() string {
	return "github"
}

func (c *GetCombinedCommitStatus) Color() string {
	return "gray"
}

func (c *GetCombinedCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCombinedCommitStatus) Configuration() []configuration.Field {
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
			Placeholder: "e.g., main, v1.2.3, or {{event.data.sha}}",
			Description: "Commit SHA, branch name, or tag name to inspect",
		},
	}
}

func (c *GetCombinedCommitStatus) Setup(ctx core.SetupContext) error {
	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *GetCombinedCommitStatus) Execute(ctx core.ExecutionContext) error {
	var config GetCombinedCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if config.Ref == "" {
		return fmt.Errorf("ref is required")
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	status, err := getCombinedStatus(context.Background(), client, config.Repository, config.Ref)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.combinedCommitStatus",
		[]any{status},
	)
}

func getCombinedStatus(ctx context.Context, client *common.Client, repository string, ref string) (*github.CombinedStatus, error) {
	opts := &github.ListOptions{PerPage: 100}
	statuses := []*github.RepoStatus{}

	var combined *github.CombinedStatus
	for {
		page, response, err := client.GetCombinedStatus(ctx, repository, ref, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get combined commit status: %w", err)
		}

		if combined == nil {
			combined = page
		}

		statuses = append(statuses, page.Statuses...)
		if response == nil || response.NextPage == 0 {
			break
		}

		opts.Page = response.NextPage
	}

	combined.Statuses = statuses
	return combined, nil
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
