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

//go:embed example_output_get_commit_status.json
var exampleOutputGetCommitStatus []byte

const CommitStatusesPayloadType = "gitlab.commitStatuses"

type GetCommitStatus struct{}

type GetCommitStatusConfiguration struct {
	Project string `mapstructure:"project"`
	SHA     string `mapstructure:"sha"`
	Ref     string `mapstructure:"ref"`
	Name    string `mapstructure:"name"`
}

func (c *GetCommitStatus) Name() string {
	return "gitlab.getCommitStatus"
}

func (c *GetCommitStatus) Label() string {
	return "Get Commit Status"
}

func (c *GetCommitStatus) Description() string {
	return "Retrieve the build/CI statuses of a GitLab commit"
}

func (c *GetCommitStatus) Documentation() string {
	return `The Get Commit Status component retrieves the build/CI statuses reported on a commit, returning the latest status per context by default.

## Use Cases

- **Gate on external checks**: Read a commit's statuses to decide whether a workflow should proceed
- **Status inspection**: Surface the reported checks for a commit or merge request head
- **Follow-up to Publish Commit Status**: Read back the status a workflow reported

## Configuration

- **Project** (required): The GitLab project containing the commit
- **Commit SHA** (required): The commit to read statuses for (supports expressions)
- **Ref** (optional): Only return statuses reported for this branch or tag
- **Name** (optional): Only return statuses with this job name

## Permissions

The connected token needs the ` + "`read_api`" + ` (or ` + "`api`" + `) scope and at least the **Reporter** role on the project.

## Output

Returns the list of commit status objects for the commit.`
}

func (c *GetCommitStatus) Icon() string {
	return "gitlab"
}

func (c *GetCommitStatus) Color() string {
	return "orange"
}

func (c *GetCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCommitStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetCommitStatus, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *GetCommitStatus) Configuration() []configuration.Field {
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
			Name:        "sha",
			Label:       "Commit SHA",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "{{ event.data.checkout_sha }}",
			Description: "The commit SHA to read statuses for. Supports expressions.",
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only return statuses reported for this branch or tag",
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only return statuses with this job name",
		},
	}
}

func (c *GetCommitStatus) Setup(ctx core.SetupContext) error {
	var config GetCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if strings.TrimSpace(config.SHA) == "" {
		return errors.New("commit SHA is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *GetCommitStatus) Execute(ctx core.ExecutionContext) error {
	var config GetCommitStatusConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.SHA) == "" {
		return errors.New("commit SHA is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	statuses, err := client.ListCommitStatuses(config.Project, strings.TrimSpace(config.SHA), &ListCommitStatusesRequest{
		Ref:  strings.TrimSpace(config.Ref),
		Name: strings.TrimSpace(config.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to get commit status: %w", err)
	}

	if statuses == nil {
		// Emit an empty array rather than JSON null when a commit has no statuses.
		statuses = []CommitStatus{}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CommitStatusesPayloadType,
		[]any{statuses},
	)
}

func (c *GetCommitStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCommitStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetCommitStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
