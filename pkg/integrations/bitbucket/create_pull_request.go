package bitbucket

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_create_pull_request.json
var exampleOutputCreatePullRequest []byte

type CreatePullRequest struct{}

type CreatePullRequestConfiguration struct {
	Repository        string   `json:"repository" mapstructure:"repository"`
	SourceBranch      string   `json:"sourceBranch" mapstructure:"sourceBranch"`
	TargetBranch      string   `json:"targetBranch" mapstructure:"targetBranch"`
	Title             string   `json:"title" mapstructure:"title"`
	Description       string   `json:"description" mapstructure:"description"`
	Reviewers         []string `json:"reviewers" mapstructure:"reviewers"`
	CloseSourceBranch bool     `json:"closeSourceBranch" mapstructure:"closeSourceBranch"`
}

func (c *CreatePullRequest) Name() string {
	return "bitbucket.createPullRequest"
}

func (c *CreatePullRequest) Label() string {
	return "Create Pull Request"
}

func (c *CreatePullRequest) Description() string {
	return "Open a new pull request in a Bitbucket repository"
}

func (c *CreatePullRequest) Documentation() string {
	return `The Create Pull Request component opens a new pull request in a Bitbucket repository.

## Use Cases

- **Automated fixes**: An agent pushes a fix to a branch and opens a pull request for human review
- **Dependency updates**: Open a pull request when a dependency bump or changelog is generated
- **Release automation**: Promote a release branch into ` + "`main`" + ` through a reviewable pull request

## Configuration

- **Repository** (required): The repository where the pull request is opened
- **Source Branch** (required): The branch holding the changes (supports expressions)
- **Target Branch** (optional): The branch to merge into. Leave empty to use the repository's main branch.
- **Title** (required): The title of the pull request
- **Description** (optional): The pull request description, in Markdown
- **Reviewers** (optional): Workspace members to request a review from
- **Close Source Branch** (optional): Delete the source branch once the pull request is merged

## Permissions

The token needs the ` + "`pullrequest:write`" + ` scope on the repository.

## Output

The component emits the created pull request, including:
- **id**: The pull request number used by every other Bitbucket component
- **state**: ` + "`OPEN`" + ` for a freshly created pull request
- **source.branch.name** and **destination.branch.name**: The branches involved
- **links.html.href**: The URL to open the pull request in Bitbucket`
}

func (c *CreatePullRequest) Icon() string {
	return "bitbucket"
}

func (c *CreatePullRequest) Color() string {
	return "blue"
}

func (c *CreatePullRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreatePullRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreatePullRequest, &example); err != nil {
		return map[string]any{}
	}

	return example
}

func (c *CreatePullRequest) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{
			Name:        "sourceBranch",
			Label:       "Source Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "feature/login-page or {{ root().data.push.changes[0].new.name }}",
			Description: "The branch that holds the changes",
		},
		{
			Name:        "targetBranch",
			Label:       "Target Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "main",
			Description: "The branch to merge into. Leave empty to use the repository's main branch.",
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "description",
			Label:    "Description",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:        "reviewers",
			Label:       "Reviewers",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Workspace members to request a review from",
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
			Description: "Delete the source branch when the pull request is merged",
		},
	}
}

func (c *CreatePullRequest) Setup(ctx core.SetupContext) error {
	config := CreatePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.SourceBranch == "" {
		return fmt.Errorf("source branch is required")
	}

	if config.Title == "" {
		return fmt.Errorf("title is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (c *CreatePullRequest) Execute(ctx core.ExecutionContext) error {
	config := CreatePullRequestConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	target, err := resolveRepositoryTarget(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	request := &CreatePullRequestRequest{
		Title:             config.Title,
		Description:       config.Description,
		Source:            PullRequestEndpoint{Branch: Branch{Name: normalizeBranch(config.SourceBranch)}},
		CloseSourceBranch: config.CloseSourceBranch,
		Reviewers:         accountRefs(config.Reviewers),
	}

	// Bitbucket falls back to the repository main branch when no destination is sent,
	// so an empty target branch stays absent from the payload rather than becoming "".
	if targetBranch := normalizeBranch(config.TargetBranch); targetBranch != "" {
		request.Destination = &PullRequestEndpoint{Branch: Branch{Name: targetBranch}}
	}

	pullRequest, err := target.Client.CreatePullRequest(target.Workspace, target.Repository, request)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"bitbucket.pullRequest",
		[]any{pullRequest},
	)
}

func (c *CreatePullRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreatePullRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreatePullRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreatePullRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreatePullRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreatePullRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// normalizeBranch accepts both a plain branch name and a full ref, because branch
// values often come from a push event where the ref is spelled refs/heads/<name>.
func normalizeBranch(branch string) string {
	return strings.TrimPrefix(strings.TrimSpace(branch), "refs/heads/")
}

func accountRefs(accounts []string) []AccountRef {
	refs := make([]AccountRef, 0, len(accounts))
	for _, account := range accounts {
		account = strings.TrimSpace(account)
		if account == "" {
			continue
		}

		refs = append(refs, AccountRef{UUID: account})
	}

	if len(refs) == 0 {
		return nil
	}

	return refs
}
