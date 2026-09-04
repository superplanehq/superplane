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

//go:embed example_output_publish_commit_status.json
var exampleOutputPublishCommitStatus []byte

type PublishCommitStatus struct{}

type PublishCommitStatusConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	Commit      string `json:"commit" mapstructure:"commit"`
	Key         string `json:"key" mapstructure:"key"`
	Name        string `json:"name" mapstructure:"name"`
	State       string `json:"state" mapstructure:"state"`
	URL         string `json:"url" mapstructure:"url"`
	Description string `json:"description" mapstructure:"description"`
}

func (p *PublishCommitStatus) Name() string {
	return "bitbucket.publishCommitStatus"
}

func (p *PublishCommitStatus) Label() string {
	return "Publish Commit Status"
}

func (p *PublishCommitStatus) Description() string {
	return "Report a build status on a Bitbucket commit"
}

func (p *PublishCommitStatus) Documentation() string {
	return `The Publish Commit Status component reports a build status on a Bitbucket commit, so the result of a
SuperPlane workflow shows up next to the commit and on any pull request that contains it.

## Use Cases

- **Make SuperPlane a required check**: Publish ` + "`INPROGRESS`" + ` when a workflow starts and ` + "`SUCCESSFUL`" + ` or ` + "`FAILED`" + ` when it finishes, then require that key in a Bitbucket branch restriction
- **Surface external results**: Report the outcome of a security scan, a load test or a manual approval directly on the commit
- **Close the loop on deploys**: Mark the commit that reached production

## Configuration

- **Repository** (required): The repository containing the commit
- **Commit** (required): The full commit hash to report on (supports expressions)
- **Key** (required): A stable identifier for this check, e.g. ` + "`superplane-deploy`" + `. Publishing again with the same key updates the existing status instead of adding a new one.
- **Name** (optional): The human-readable name shown in Bitbucket. Defaults to the key.
- **State** (required): The build state to report
- **URL** (optional): A link to the run, so people can click through from Bitbucket
- **Description** (optional): A short explanation shown next to the status

## Permissions

The token needs the ` + "`repository:write`" + ` scope on the repository.

## Output

The component emits the published status, including ` + "`key`" + `, ` + "`state`" + `, ` + "`url`" + ` and ` + "`refname`" + `.

Publishing a status fires the **On Commit Status** trigger, so be careful not to point that trigger at the
same key this component writes — it would run the workflow again.`
}

func (p *PublishCommitStatus) Icon() string {
	return "bitbucket"
}

func (p *PublishCommitStatus) Color() string {
	return "blue"
}

func (p *PublishCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (p *PublishCommitStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputPublishCommitStatus, &example); err != nil {
		return map[string]any{}
	}

	return example
}

func (p *PublishCommitStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		commitField(),
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "superplane-deploy",
			Description: "A stable identifier for this check. Publishing again with the same key updates the existing status.",
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The name shown in Bitbucket. Defaults to the key.",
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  StateInProgress,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: commitStatusStates,
				},
			},
		},
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "A link people can follow from Bitbucket back to this run",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "A short explanation shown next to the status",
		},
	}
}

func (p *PublishCommitStatus) Setup(ctx core.SetupContext) error {
	config := PublishCommitStatusConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Commit == "" {
		return fmt.Errorf("commit is required")
	}

	if config.Key == "" {
		return fmt.Errorf("key is required")
	}

	if config.State == "" {
		return fmt.Errorf("state is required")
	}

	if !isValidCommitStatusState(config.State) {
		return fmt.Errorf("unsupported build state %q", config.State)
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (p *PublishCommitStatus) Execute(ctx core.ExecutionContext) error {
	config := PublishCommitStatusConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	commit := strings.TrimSpace(config.Commit)
	if commit == "" {
		return fmt.Errorf("commit is required")
	}

	target, err := resolveRepositoryTarget(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	name := config.Name
	if name == "" {
		name = config.Key
	}

	published, err := target.Client.PublishCommitStatus(target.Workspace, target.Repository, commit, &CommitStatus{
		Key:         config.Key,
		Name:        name,
		State:       strings.ToUpper(strings.TrimSpace(config.State)),
		URL:         config.URL,
		Description: config.Description,
	})

	if err != nil {
		return fmt.Errorf("failed to publish commit status: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"bitbucket.commitStatus",
		[]any{published},
	)
}

func (p *PublishCommitStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (p *PublishCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (p *PublishCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (p *PublishCommitStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (p *PublishCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *PublishCommitStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
