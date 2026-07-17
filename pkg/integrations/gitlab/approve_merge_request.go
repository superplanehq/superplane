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

//go:embed example_output_approve_merge_request.json
var exampleOutputApproveMergeRequest []byte

type ApproveMergeRequest struct{}

type ApproveMergeRequestConfiguration struct {
	Project         string `mapstructure:"project"`
	MergeRequestIID string `mapstructure:"mergeRequestIid"`
	SHA             string `mapstructure:"sha"`
}

func (c *ApproveMergeRequest) Name() string {
	return "gitlab.approveMergeRequest"
}

func (c *ApproveMergeRequest) Label() string {
	return "Approve Merge Request"
}

func (c *ApproveMergeRequest) Description() string {
	return "Approve a GitLab merge request"
}

func (c *ApproveMergeRequest) Documentation() string {
	return `The Approve Merge Request component approves a GitLab merge request as the connected user.

## Use Cases

- **Automated approval gates**: Approve a merge request after agent checks, review pipelines, or policy checks pass
- **Bot reviews**: Approve merge requests from a controlled SuperPlane workflow
- **Compliance workflows**: Record an approval once external validations succeed

## Configuration

- **Project** (required): The GitLab project containing the merge request
- **Merge Request IID** (required): The internal ID (IID) of the merge request to approve (supports expressions)
- **Expected SHA**: Optional head SHA guard. GitLab rejects the approval if the source branch head has changed.

## Permissions

The connected user must be an eligible approver on the project: a direct member with
at least the **Developer** role. Merge request authors cannot approve their own merge
requests unless the project allows it.

## Output

Returns the merge request approval state, including the number of approvals required and left, and who approved.`
}

func (c *ApproveMergeRequest) Icon() string {
	return "gitlab"
}

func (c *ApproveMergeRequest) Color() string {
	return "orange"
}

func (c *ApproveMergeRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ApproveMergeRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputApproveMergeRequest, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *ApproveMergeRequest) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the merge request to approve",
		},
		{
			Name:        "sha",
			Label:       "Expected SHA",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "{{event.data.object_attributes.last_commit.id}}",
			Description: "Optional source branch head SHA. GitLab rejects the approval if the head changed.",
		},
	}
}

func (c *ApproveMergeRequest) Setup(ctx core.SetupContext) error {
	var config ApproveMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.MergeRequestIID == "" {
		return fmt.Errorf("merge request IID is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *ApproveMergeRequest) Execute(ctx core.ExecutionContext) error {
	var config ApproveMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	approval, err := client.ApproveMergeRequest(context.Background(), config.Project, config.MergeRequestIID, &ApproveMergeRequestRequest{
		SHA: config.SHA,
	})
	if err != nil {
		return fmt.Errorf("failed to approve merge request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequestApproval",
		[]any{approval},
	)
}

func (c *ApproveMergeRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ApproveMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *ApproveMergeRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ApproveMergeRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ApproveMergeRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ApproveMergeRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
