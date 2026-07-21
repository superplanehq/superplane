package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_mark_merge_request_ready_for_review.json
var exampleOutputMarkMergeRequestReadyForReview []byte

// readyQuickAction is the GitLab quick action that clears a merge request's
// draft status. There is no dedicated field for this on the Update MR REST
// endpoint, so it is applied the same way the GitLab UI does it: as a note.
// See https://docs.gitlab.com/user/project/quick_actions/
const readyQuickAction = "/ready"

type MarkMergeRequestReadyForReview struct{}

type MarkMergeRequestReadyForReviewConfiguration struct {
	Project         string `mapstructure:"project"`
	MergeRequestIID string `mapstructure:"mergeRequestIid"`
}

func (c *MarkMergeRequestReadyForReview) Name() string {
	return "gitlab.markMergeRequestReadyForReview"
}

func (c *MarkMergeRequestReadyForReview) Label() string {
	return "Mark Merge Request Ready for Review"
}

func (c *MarkMergeRequestReadyForReview) Description() string {
	return "Take a draft merge request out of the draft state in a GitLab project"
}

func (c *MarkMergeRequestReadyForReview) Documentation() string {
	return `The Mark Merge Request Ready for Review component takes a draft merge request out of the draft state, the same as clicking "Mark as ready" on GitLab.

## Use Cases

- **Promote drafts automatically**: Mark a draft merge request ready once CI checks pass
- **Release trains**: Open drafts early and promote them for review when the branch is ready
- **Bot workflows**: Let an automation open work as a draft and hand it to reviewers when complete

## Configuration

- **Project**: Select the GitLab project containing the merge request
- **Merge Request IID**: The internal ID (IID) of the merge request to mark ready for review. Expressions are supported.

## Behavior

This component is idempotent: if the merge request is already out of the draft state, it succeeds without calling GitLab again and emits the merge request as it is.

## Permissions

GitLab does not expose a dedicated REST field for clearing draft status, so the component applies the same ` + "`/ready`" + ` quick action the GitLab UI uses. This is submitted as a note on the merge request, so the connected user needs permission to comment on it.

## Output

Returns the merge request object after it has been marked ready for review.`
}

func (c *MarkMergeRequestReadyForReview) Icon() string {
	return "gitlab"
}

func (c *MarkMergeRequestReadyForReview) Color() string {
	return "orange"
}

func (c *MarkMergeRequestReadyForReview) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *MarkMergeRequestReadyForReview) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputMarkMergeRequestReadyForReview, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *MarkMergeRequestReadyForReview) Configuration() []configuration.Field {
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
			Name:        "mergeRequestIid",
			Label:       "Merge Request IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "42 or {{event.data.object_attributes.iid}}",
			Description: "The internal ID (IID) of the merge request to mark ready for review",
		},
	}
}

func (c *MarkMergeRequestReadyForReview) Setup(ctx core.SetupContext) error {
	var config MarkMergeRequestReadyForReviewConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.MergeRequestIID == "" {
		return errors.New("merge request IID is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *MarkMergeRequestReadyForReview) Execute(ctx core.ExecutionContext) error {
	var config MarkMergeRequestReadyForReviewConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.MergeRequestIID == "" {
		return errors.New("merge request IID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	mergeRequest, err := client.GetMergeRequest(context.Background(), config.Project, config.MergeRequestIID)
	if err != nil {
		return fmt.Errorf("failed to get merge request: %w", err)
	}

	//
	// Applying the /ready quick action to an already-ready merge request is a
	// no-op on GitLab's side, but it is skipped here to avoid an unnecessary
	// note and to keep re-runs idempotent.
	//
	if mergeRequest.Draft {
		_, err = client.CreateMergeRequestNote(context.Background(), config.Project, config.MergeRequestIID, &CreateNoteRequest{
			Body: readyQuickAction,
		})
		if err != nil {
			return fmt.Errorf("failed to mark merge request ready for review: %w", err)
		}

		mergeRequest, err = client.GetMergeRequest(context.Background(), config.Project, config.MergeRequestIID)
		if err != nil {
			return fmt.Errorf("failed to get merge request: %w", err)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequest",
		[]any{mergeRequest},
	)
}

func (c *MarkMergeRequestReadyForReview) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *MarkMergeRequestReadyForReview) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *MarkMergeRequestReadyForReview) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *MarkMergeRequestReadyForReview) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *MarkMergeRequestReadyForReview) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *MarkMergeRequestReadyForReview) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
