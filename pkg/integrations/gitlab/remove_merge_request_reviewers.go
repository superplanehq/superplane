package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_remove_merge_request_reviewers.json
var exampleOutputRemoveMergeRequestReviewers []byte

type RemoveMergeRequestReviewers struct{}

type RemoveMergeRequestReviewersConfiguration struct {
	Project         string   `mapstructure:"project"`
	MergeRequestIID string   `mapstructure:"mergeRequestIid"`
	Reviewers       []string `mapstructure:"reviewers"`
}

func (c *RemoveMergeRequestReviewers) Name() string {
	return "gitlab.removeMergeRequestReviewers"
}

func (c *RemoveMergeRequestReviewers) Label() string {
	return "Remove Merge Request Reviewers"
}

func (c *RemoveMergeRequestReviewers) Description() string {
	return "Remove reviewers from a GitLab merge request"
}

func (c *RemoveMergeRequestReviewers) Documentation() string {
	return `The Remove Merge Request Reviewers component removes reviewers from an existing GitLab merge request. Reviewers that are not listed are kept.

## Use Cases

- **Automated cleanup**: Remove a reviewer once their review is no longer needed
- **Reassignment**: Remove a reviewer before adding a different one as part of a workflow
- **Rotation**: Drop an out-of-office reviewer from an open merge request

## Configuration

- **Project** (required): The GitLab project containing the merge request
- **Merge Request IID** (required): The internal ID (IID) of the merge request (supports expressions)
- **Reviewers** (required): Users to remove from the merge request's reviewers. Reviewers not listed are kept.

## Permissions

The connected user needs at least the **Developer** role on the project to change reviewers.

## Output

Returns the updated merge request object, including its remaining reviewer list and URL.`
}

func (c *RemoveMergeRequestReviewers) Icon() string {
	return "gitlab"
}

func (c *RemoveMergeRequestReviewers) Color() string {
	return "orange"
}

func (c *RemoveMergeRequestReviewers) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveMergeRequestReviewers) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputRemoveMergeRequestReviewers, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *RemoveMergeRequestReviewers) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the merge request",
		},
		{
			Name:     "reviewers",
			Label:    "Reviewers",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeMember,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
}

func (c *RemoveMergeRequestReviewers) Setup(ctx core.SetupContext) error {
	var config RemoveMergeRequestReviewersConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.MergeRequestIID == "" {
		return fmt.Errorf("merge request IID is required")
	}

	if len(config.Reviewers) == 0 {
		return fmt.Errorf("at least one reviewer is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *RemoveMergeRequestReviewers) Execute(ctx core.ExecutionContext) error {
	var config RemoveMergeRequestReviewersConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	mergeRequest, err := client.GetMergeRequest(context.Background(), config.Project, config.MergeRequestIID)
	if err != nil {
		return fmt.Errorf("failed to get merge request: %w", err)
	}

	reviewerIDs := removeReviewerIDs(reviewerIDsOf(mergeRequest), parseUserIDs(config.Reviewers))

	updated, err := client.UpdateMergeRequestReviewers(context.Background(), config.Project, config.MergeRequestIID, reviewerIDs)
	if err != nil {
		return fmt.Errorf("failed to remove merge request reviewers: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequest",
		[]any{updated},
	)
}

func (c *RemoveMergeRequestReviewers) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveMergeRequestReviewers) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *RemoveMergeRequestReviewers) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveMergeRequestReviewers) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RemoveMergeRequestReviewers) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RemoveMergeRequestReviewers) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
