package claude

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const MergeAgentPayloadType = "claude.mergeAgent.finished"

type MergeAgent struct{}

type MergeAgentSpec struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	PrNumber    string `json:"prNumber" mapstructure:"prNumber"`
	IssueKey    string `json:"issueKey" mapstructure:"issueKey"`
	GitHubToken string `json:"githubToken" mapstructure:"githubToken"`
}

type MergeAgentOutputPayload struct {
	Status   string `json:"status"`
	Branch   string `json:"branch"`
	Message  string `json:"message"`
	ErrorMsg string `json:"error,omitempty"`
}

func (m *MergeAgent) Name() string  { return "claude.mergeAgent" }
func (m *MergeAgent) Label() string { return "Merge PR" }

func (m *MergeAgent) Description() string {
	return "Merges an approved pull request into the base branch"
}

func (m *MergeAgent) Documentation() string {
	return `The Merge PR component merges an approved pull request into its base branch using squash merge.

## Use Cases
- **Auto-merge on approval**: Merge the PR after Claude review approves it
- **Release flow**: Merge feature branch to main after QA

## Configuration
- **Repository**: GitHub repository in owner/repo format
- **PR Number**: Pull request number to merge
- **Issue Key**: Used in the commit message
- **GitHub Token**: Personal access token with repo scope

## Output
- **status**: succeeded or failed
- **branch**: The branch that was merged
- **message**: Commit message used for the merge`
}

func (m *MergeAgent) Icon() string  { return "git-merge" }
func (m *MergeAgent) Color() string { return "green" }

func (m *MergeAgent) ExampleOutput() map[string]any {
	return map[string]any{
		"type": MergeAgentPayloadType,
		"data": map[string]any{
			"status":  "succeeded",
			"branch":  "KAN-1",
			"message": "feat(KAN-1): merged via AI Delivery Orchestrator",
		},
	}
}

func (m *MergeAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (m *MergeAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "GitHub repository in owner/repo format",
			Placeholder: "acme/my-app",
		},
		{
			Name:        "prNumber",
			Label:       "PR Number",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Pull request number to merge",
		},
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Used in the merge commit message",
			Placeholder: "KAN-1",
		},
		{
			Name:        "githubToken",
			Label:       "GitHub Token",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "GitHub personal access token with repo scope",
		},
	}
}

func (m *MergeAgent) Setup(ctx core.SetupContext) error {
	spec := MergeAgentSpec{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{WeaklyTypedInput: true, Result: &spec})
	if err := decoder.Decode(ctx.Configuration); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}
	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if spec.GitHubToken == "" {
		return fmt.Errorf("githubToken is required")
	}
	return nil
}

func (m *MergeAgent) Execute(ctx core.ExecutionContext) error {
	spec := MergeAgentSpec{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{WeaklyTypedInput: true, Result: &spec})
	if err := decoder.Decode(ctx.Configuration); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	parts := strings.SplitN(spec.Repository, "/", 2)
	if len(parts) != 2 {
		return emitMergeFailure(ctx, "repository must be in owner/repo format")
	}
	owner, repo := parts[0], parts[1]

	prNumber, err := strconv.Atoi(spec.PrNumber)
	if err != nil {
		return emitMergeFailure(ctx, fmt.Sprintf("invalid prNumber %q: %v", spec.PrNumber, err))
	}

	commitMsg := fmt.Sprintf("feat(%s): merged via AI Delivery Orchestrator", spec.IssueKey)

	ghClient := NewGitHubClient(spec.GitHubToken, ctx.HTTP)

	ctx.Logger.Infof("[MergeAgent] Merging PR #%d from %s", prNumber, spec.Repository)
	if err := ghClient.MergePR(owner, repo, prNumber, commitMsg); err != nil {
		return emitMergeFailure(ctx, fmt.Sprintf("failed to merge PR: %v", err))
	}

	ctx.Logger.Infof("[MergeAgent] PR #%d merged successfully", prNumber)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		MergeAgentPayloadType,
		[]any{MergeAgentOutputPayload{
			Status:  "succeeded",
			Branch:  spec.IssueKey,
			Message: commitMsg,
		}},
	)
}

func emitMergeFailure(ctx core.ExecutionContext, errMsg string) error {
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		MergeAgentPayloadType,
		[]any{MergeAgentOutputPayload{
			Status:   "failed",
			ErrorMsg: errMsg,
		}},
	)
}

func (m *MergeAgent) Cancel(ctx core.ExecutionContext) error      { return nil }
func (m *MergeAgent) Cleanup(ctx core.SetupContext) error          { return nil }
func (m *MergeAgent) Actions() []core.Action                       { return []core.Action{} }
func (m *MergeAgent) HandleAction(ctx core.ActionContext) error    { return nil }

func (m *MergeAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (m *MergeAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
