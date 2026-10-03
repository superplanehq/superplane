package bitbucket

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_merge_pull_request.json
var exampleOutputMergePullRequest []byte

type MergePullRequest struct{}

type MergePullRequestConfiguration struct {
	Repository        string `json:"repository" mapstructure:"repository"`
	PullRequestID     string `json:"pullRequestId" mapstructure:"pullRequestId"`
	MergeStrategy     string `json:"mergeStrategy" mapstructure:"mergeStrategy"`
	Message           string `json:"message" mapstructure:"message"`
	CloseSourceBranch bool   `json:"closeSourceBranch" mapstructure:"closeSourceBranch"`
}

func (m *MergePullRequest) Name() string {
	return "bitbucket.mergePullRequest"
}

func (m *MergePullRequest) Label() string {
	return "Merge Pull Request"
}

func (m *MergePullRequest) Description() string {
	return "Merge an open pull request in a Bitbucket repository"
}

func (m *MergePullRequest) Documentation() string {
	return `The Merge Pull Request component merges an open Bitbucket pull request.

## Use Cases

- **Policy-gated merges**: Merge only after approvals, a green build and a manual approval step have all passed
- **Auto-merge**: Merge dependency or changelog pull requests once CI reports success
- **Release trains**: Merge a batch of ready pull requests as part of a coordinated release

## Configuration

- **Repository** (required): The repository containing the pull request
- **Pull Request ID** (required): The numeric ID of the pull request (supports expressions)
- **Merge Strategy** (required): How the merge is performed. Default: Merge commit.
- **Message** (optional): The merge commit message. Leave empty to let Bitbucket generate it.
- **Close Source Branch** (optional): Delete the source branch after merging

### Merge strategies

| Strategy | Behaviour |
| --- | --- |
| **Merge commit** | Creates a merge commit, keeping the full branch history |
| **Squash** | Combines every commit on the branch into a single commit |
| **Fast forward** | Moves the target branch pointer, and fails if the branch has diverged |

## Permissions

The token needs the ` + "`pullrequest:write`" + ` scope, and the account must be allowed to write to the
target branch. Branch restrictions, unresolved merge checks and merge conflicts all make this component fail
with the reason reported by Bitbucket.

## Output

The component emits the merged pull request, including:
- **state**: ` + "`MERGED`" + `
- **merge_commit.hash**: The commit created by the merge
- **closed_by**: The account the merge was performed as

If Bitbucket queues the merge asynchronously — which happens on very large repositories — the component
fails rather than reporting a merge that has not completed.`
}

func (m *MergePullRequest) Icon() string {
	return "bitbucket"
}

func (m *MergePullRequest) Color() string {
	return "blue"
}

func (m *MergePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (m *MergePullRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputMergePullRequest, &example); err != nil {
		return map[string]any{}
	}

	return example
}

func (m *MergePullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		pullRequestIDField(),
		{
			Name:     "mergeStrategy",
			Label:    "Merge Strategy",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  MergeStrategyMergeCommit,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Merge commit", Value: MergeStrategyMergeCommit},
						{Label: "Squash", Value: MergeStrategySquash},
						{Label: "Fast forward", Value: MergeStrategyFastForward},
					},
				},
			},
		},
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "The merge commit message. Leave empty to let Bitbucket generate it.",
		},
		{
			Name:        "closeSourceBranch",
			Label:       "Close Source Branch",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Delete the source branch after merging",
		},
	}
}

func (m *MergePullRequest) Setup(ctx core.SetupContext) error {
	config := MergePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.PullRequestID == "" {
		return fmt.Errorf("pull request ID is required")
	}

	switch config.MergeStrategy {
	case "", MergeStrategyMergeCommit, MergeStrategySquash, MergeStrategyFastForward:
	default:
		return fmt.Errorf("unsupported merge strategy %q", config.MergeStrategy)
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (m *MergePullRequest) Execute(ctx core.ExecutionContext) error {
	config := MergePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	pullRequestID, err := parsePullRequestID(config.PullRequestID)
	if err != nil {
		return err
	}

	target, err := resolveRepositoryTarget(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	request := &MergePullRequestRequest{
		Type:          "pullrequest",
		Message:       config.Message,
		MergeStrategy: config.MergeStrategy,
	}

	if config.CloseSourceBranch {
		request.CloseSourceBranch = &config.CloseSourceBranch
	}

	pullRequest, err := target.Client.MergePullRequest(target.Workspace, target.Repository, pullRequestID, request)
	if err != nil {
		return fmt.Errorf("failed to merge pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"bitbucket.pullRequest",
		[]any{pullRequest},
	)
}

func (m *MergePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (m *MergePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (m *MergePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (m *MergePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (m *MergePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (m *MergePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
