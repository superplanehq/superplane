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

//go:embed example_output_update_pull_request.json
var exampleOutputUpdatePullRequest []byte

type UpdatePullRequest struct{}

type UpdatePullRequestConfiguration struct {
	Repository        string   `json:"repository" mapstructure:"repository"`
	PullRequestID     string   `json:"pullRequestId" mapstructure:"pullRequestId"`
	Title             string   `json:"title" mapstructure:"title"`
	Description       string   `json:"description" mapstructure:"description"`
	TargetBranch      string   `json:"targetBranch" mapstructure:"targetBranch"`
	Reviewers         []string `json:"reviewers" mapstructure:"reviewers"`
	CloseSourceBranch bool     `json:"closeSourceBranch" mapstructure:"closeSourceBranch"`
}

// updatePullRequestToggles tracks which optional fields were explicitly turned on via
// their UI toggle, independent of whether the decoded value ended up empty - clearing
// a field (toggled on, empty value) must still be sent, unlike a field that was never
// toggled on. See gitlab/update_merge_request.go for the same pattern.
type updatePullRequestToggles struct {
	Title             bool
	Description       bool
	TargetBranch      bool
	Reviewers         bool
	CloseSourceBranch bool
}

func newUpdatePullRequestToggles(raw map[string]any) updatePullRequestToggles {
	enabled := func(field string) bool {
		value, ok := raw[field]
		return ok && value != nil
	}

	return updatePullRequestToggles{
		Title:             enabled("title"),
		Description:       enabled("description"),
		TargetBranch:      enabled("targetBranch"),
		Reviewers:         enabled("reviewers"),
		CloseSourceBranch: enabled("closeSourceBranch"),
	}
}

func (t updatePullRequestToggles) hasUpdates() bool {
	return t.Title || t.Description || t.TargetBranch || t.Reviewers || t.CloseSourceBranch
}

func (u *UpdatePullRequest) Name() string {
	return "bitbucket.updatePullRequest"
}

func (u *UpdatePullRequest) Label() string {
	return "Update Pull Request"
}

func (u *UpdatePullRequest) Description() string {
	return "Update an existing pull request in a Bitbucket repository"
}

func (u *UpdatePullRequest) Documentation() string {
	return `The Update Pull Request component modifies an existing Bitbucket pull request: its title, description, target branch, reviewers, or whether the source branch is deleted on merge.

## Use Cases

- **Progress reporting**: Rewrite the description as a workflow advances, so the pull request always shows current state
- **Retargeting**: Move a pull request onto a different release branch
- **Review routing**: Assign reviewers based on which files changed

## Configuration

- **Repository** (required): The repository containing the pull request
- **Pull Request ID** (required): The numeric ID of the pull request (supports expressions)
- **Title** (toggle): New title for the pull request
- **Description** (toggle): New description for the pull request
- **Target Branch** (toggle): Retarget the pull request onto a different branch
- **Reviewers** (toggle): Workspace members to request a review from. Replaces the existing reviewers; selecting none clears them.
- **Close Source Branch** (toggle): Whether the source branch is deleted when the pull request merges

Each field besides Repository and Pull Request ID is toggled on individually, so only the fields you enable are sent. At least one must be enabled. Bitbucket rejects a blank title, so Title must have a value when enabled.

## Permissions

The token needs the ` + "`pullrequest:write`" + ` scope on the repository.

## Output

The component emits the updated pull request, in the same shape as **Create Pull Request**.`
}

func (u *UpdatePullRequest) Icon() string {
	return "bitbucket"
}

func (u *UpdatePullRequest) Color() string {
	return "blue"
}

func (u *UpdatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdatePullRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputUpdatePullRequest, &example); err != nil {
		return map[string]any{}
	}

	return example
}

func (u *UpdatePullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		pullRequestIDField(),
		{
			Name:      "title",
			Label:     "Title",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "description",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:        "targetBranch",
			Label:       "Target Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Retarget the pull request onto a different branch",
		},
		{
			Name:        "reviewers",
			Label:       "Reviewers",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Description: "Workspace members to request a review from. Replaces the existing reviewers; selecting none clears them.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeMember,
					Multi: true,
				},
			},
		},
		{
			Name:        "closeSourceBranch",
			Label:       "Close Source Branch",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Description: "Whether the source branch is deleted when the pull request merges",
		},
	}
}

func (u *UpdatePullRequest) Setup(ctx core.SetupContext) error {
	config := UpdatePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.PullRequestID == "" {
		return fmt.Errorf("pull request ID is required")
	}

	raw, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return fmt.Errorf("failed to decode configuration")
	}

	toggles := newUpdatePullRequestToggles(raw)
	if !toggles.hasUpdates() {
		return fmt.Errorf("at least one field to update is required")
	}

	// Bitbucket rejects an empty title, so an enabled-but-blank title is a
	// configuration error rather than a request we should let fail at runtime.
	if toggles.Title && config.Title == "" {
		return fmt.Errorf("title cannot be empty when it is enabled")
	}

	if toggles.TargetBranch && normalizeBranch(config.TargetBranch) == "" {
		return fmt.Errorf("target branch cannot be empty when it is enabled")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (u *UpdatePullRequest) Execute(ctx core.ExecutionContext) error {
	config := UpdatePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	raw, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return fmt.Errorf("failed to decode configuration")
	}

	pullRequestID, err := parsePullRequestID(config.PullRequestID)
	if err != nil {
		return err
	}

	target, err := resolveRepositoryTarget(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	toggles := newUpdatePullRequestToggles(raw)
	request := &UpdatePullRequestRequest{}

	if toggles.Title {
		request.Title = &config.Title
	}

	if toggles.Description {
		request.Description = &config.Description
	}

	if toggles.TargetBranch {
		request.Destination = &PullRequestEndpoint{Branch: Branch{Name: normalizeBranch(config.TargetBranch)}}
	}

	if toggles.Reviewers {
		reviewers := accountRefs(config.Reviewers)
		if reviewers == nil {
			reviewers = []AccountRef{}
		}

		request.Reviewers = &reviewers
	}

	if toggles.CloseSourceBranch {
		request.CloseSourceBranch = &config.CloseSourceBranch
	}

	pullRequest, err := target.Client.UpdatePullRequest(target.Workspace, target.Repository, pullRequestID, request)
	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"bitbucket.pullRequest",
		[]any{pullRequest},
	)
}

func (u *UpdatePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdatePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (u *UpdatePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdatePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (u *UpdatePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdatePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
