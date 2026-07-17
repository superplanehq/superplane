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

//go:embed example_output_add_merge_request_reviewers.json
var exampleOutputAddMergeRequestReviewers []byte

type AddMergeRequestReviewers struct{}

type AddMergeRequestReviewersConfiguration struct {
	Project         string   `mapstructure:"project"`
	MergeRequestIID string   `mapstructure:"mergeRequestIid"`
	Reviewers       []string `mapstructure:"reviewers"`
}

func (c *AddMergeRequestReviewers) Name() string {
	return "gitlab.addMergeRequestReviewers"
}

func (c *AddMergeRequestReviewers) Label() string {
	return "Add Merge Request Reviewers"
}

func (c *AddMergeRequestReviewers) Description() string {
	return "Add reviewers to a GitLab merge request"
}

func (c *AddMergeRequestReviewers) Documentation() string {
	return `The Add Merge Request Reviewers component requests reviews from additional users on an existing GitLab merge request. Existing reviewers are kept.

## Use Cases

- **Automated review assignment**: Add reviewers after a merge request is opened as part of a workflow
- **Escalation**: Add a senior reviewer when checks fail or a change touches sensitive areas
- **Round-robin**: Assign reviewers from a rotation once a merge request is ready

## Configuration

- **Project** (required): The GitLab project containing the merge request
- **Merge Request IID** (required): The internal ID (IID) of the merge request (supports expressions)
- **Reviewers** (required): Users to request a review from. These are added to any existing reviewers.

## Permissions

The connected user needs at least the **Developer** role on the project to change reviewers.

## Output

Returns the updated merge request object, including its full reviewer list and URL.`
}

func (c *AddMergeRequestReviewers) Icon() string {
	return "gitlab"
}

func (c *AddMergeRequestReviewers) Color() string {
	return "orange"
}

func (c *AddMergeRequestReviewers) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddMergeRequestReviewers) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputAddMergeRequestReviewers, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *AddMergeRequestReviewers) Configuration() []configuration.Field {
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

func (c *AddMergeRequestReviewers) Setup(ctx core.SetupContext) error {
	var config AddMergeRequestReviewersConfiguration
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

func (c *AddMergeRequestReviewers) Execute(ctx core.ExecutionContext) error {
	var config AddMergeRequestReviewersConfiguration
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

	reviewerIDs := mergeReviewerIDs(reviewerIDsOf(mergeRequest), parseUserIDs(config.Reviewers))

	updated, err := client.UpdateMergeRequestReviewers(context.Background(), config.Project, config.MergeRequestIID, reviewerIDs)
	if err != nil {
		return fmt.Errorf("failed to add merge request reviewers: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequest",
		[]any{updated},
	)
}

func (c *AddMergeRequestReviewers) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddMergeRequestReviewers) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AddMergeRequestReviewers) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddMergeRequestReviewers) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddMergeRequestReviewers) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddMergeRequestReviewers) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
