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

//go:embed example_output_get_commit_status.json
var exampleOutputGetCommitStatus []byte

// StateNoStatus is reported when a commit carries no build status at all. Bitbucket
// has no such value; it exists so a gate can tell "nothing ran" apart from "it passed".
const StateNoStatus = "NO_STATUS"

type GetCommitStatus struct{}

type GetCommitStatusConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Commit     string `json:"commit" mapstructure:"commit"`
}

// CombinedCommitStatus rolls every build status on a commit into a single verdict, so a
// workflow can gate on one field instead of walking the list itself.
type CombinedCommitStatus struct {
	Commit     string         `json:"commit"`
	State      string         `json:"state"`
	TotalCount int            `json:"total_count"`
	Statuses   []CommitStatus `json:"statuses"`
}

func (g *GetCommitStatus) Name() string {
	return "bitbucket.getCommitStatus"
}

func (g *GetCommitStatus) Label() string {
	return "Get Commit Status"
}

func (g *GetCommitStatus) Description() string {
	return "Read the build statuses reported on a Bitbucket commit"
}

func (g *GetCommitStatus) Documentation() string {
	return `The Get Commit Status component reads every build status reported on a commit and rolls them into a
single verdict, so a workflow can decide whether a commit is safe to ship.

## Use Cases

- **Deploy gates**: Check that every build on the commit is green before promoting it
- **Release readiness**: Verify a release candidate has passed all required checks
- **Diagnostics**: Attach the list of failing checks to an incident or a pull request comment

## Configuration

- **Repository** (required): The repository containing the commit
- **Commit** (required): The full commit hash to read (supports expressions)

## Output

The component emits an aggregate over the commit's build statuses:
- **commit**: The commit that was read
- **state**: The combined verdict, see the table below
- **total_count**: How many statuses were reported on the commit
- **statuses**: Every individual status, in the order Bitbucket returned them

### How the combined state is decided

The combined state is the most severe individual state, in this order:

| Combined state | When |
| --- | --- |
| ` + "`FAILED`" + ` | At least one status failed |
| ` + "`STOPPED`" + ` | No failures, but at least one build was stopped |
| ` + "`INPROGRESS`" + ` | Nothing failed or stopped, but at least one build is still running |
| ` + "`SUCCESSFUL`" + ` | Every status succeeded |
| ` + "`NO_STATUS`" + ` | The commit carries no build status at all |

` + "`NO_STATUS`" + ` is not a Bitbucket value — it exists so a gate can tell "nothing ran yet" apart from
"everything passed", which are very different answers when deciding to deploy.

## Permissions

The token needs the ` + "`repository`" + ` read scope.`
}

func (g *GetCommitStatus) Icon() string {
	return "bitbucket"
}

func (g *GetCommitStatus) Color() string {
	return "blue"
}

func (g *GetCommitStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetCommitStatus) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputGetCommitStatus, &example); err != nil {
		return map[string]any{}
	}

	return example
}

func (g *GetCommitStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		commitField(),
	}
}

func (g *GetCommitStatus) Setup(ctx core.SetupContext) error {
	config := GetCommitStatusConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Commit == "" {
		return fmt.Errorf("commit is required")
	}

	_, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	return err
}

func (g *GetCommitStatus) Execute(ctx core.ExecutionContext) error {
	config := GetCommitStatusConfiguration{}
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

	statuses, err := target.Client.ListCommitStatuses(target.Workspace, target.Repository, commit)
	if err != nil {
		return fmt.Errorf("failed to get commit statuses: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"bitbucket.commitStatuses",
		[]any{
			CombinedCommitStatus{
				Commit:     commit,
				State:      combineCommitStatusStates(statuses),
				TotalCount: len(statuses),
				Statuses:   statuses,
			},
		},
	)
}

func (g *GetCommitStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetCommitStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (g *GetCommitStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetCommitStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (g *GetCommitStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (g *GetCommitStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

// combineCommitStatusStates reports the most severe state on the commit. A stopped
// build ranks above a running one because a cancelled build will never turn green,
// and an unknown state is treated as a failure rather than silently passing a gate.
func combineCommitStatusStates(statuses []CommitStatus) string {
	if len(statuses) == 0 {
		return StateNoStatus
	}

	var stopped, inProgress bool

	for _, status := range statuses {
		switch strings.ToUpper(strings.TrimSpace(status.State)) {
		case StateSuccessful:
		case StateStopped:
			stopped = true
		case StateInProgress:
			inProgress = true
		default:
			return StateFailed
		}
	}

	switch {
	case stopped:
		return StateStopped
	case inProgress:
		return StateInProgress
	default:
		return StateSuccessful
	}
}
