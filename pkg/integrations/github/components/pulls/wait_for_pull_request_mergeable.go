package pulls

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	WaitForPullRequestMergeableName = "github.waitForPullRequestMergeable"

	waitMergeableEvaluateHook = "evaluate"

	waitMergeableCleanChannel      = "clean"
	waitMergeableConflictedChannel = "conflicted"
	waitMergeableTimedOutChannel   = "timedOut"

	waitMergeableDefaultTimeoutSeconds = 300
	waitMergeablePollInterval          = 10 * time.Second

	mergeableStateDirty = "dirty"
)

type WaitForPullRequestMergeable struct{}

type WaitForPullRequestMergeableConfiguration struct {
	Repository     string `mapstructure:"repository" json:"repository"`
	Number         any    `mapstructure:"number" json:"number"`
	TimeoutSeconds *int   `mapstructure:"timeoutSeconds" json:"timeoutSeconds"`
}

// WaitForPullRequestMergeableMetadata persists only the fixed poll deadline.
// Repository, number, and the timeout budget are read fresh from the node
// configuration on every evaluation, same as the initial Execute call.
type WaitForPullRequestMergeableMetadata struct {
	TimeoutAtUnix int64 `json:"timeoutAtUnix" mapstructure:"timeoutAtUnix"`
}

type WaitForPullRequestMergeableOutput struct {
	Repository     string `json:"repository"`
	Number         int    `json:"number"`
	SHA            string `json:"sha"`
	BaseRef        string `json:"baseRef"`
	Mergeable      *bool  `json:"mergeable"`
	MergeableState string `json:"mergeableState"`
}

func (c *WaitForPullRequestMergeable) Name() string {
	return WaitForPullRequestMergeableName
}

func (c *WaitForPullRequestMergeable) Label() string {
	return "Wait For Pull Request Mergeable"
}

func (c *WaitForPullRequestMergeable) Description() string {
	return "Wait until GitHub reports whether a pull request has a merge conflict"
}

func (c *WaitForPullRequestMergeable) Documentation() string {
	return `The Wait For Pull Request Mergeable component polls one pull request until GitHub finishes computing mergeability.

GitHub computes ` + "`mergeable`" + ` asynchronously after a push. The field is often ` + "`null`" + ` for a short time after a pull request changes. This component polls on a short interval until GitHub reports a definite answer, or until the timeout elapses.

## Use Cases

- **Conflict repair**: Start an agent run only when GitHub reports a real merge conflict, not a pending computation
- **Base branch rechecks**: After a push to a shared base branch, recheck every open pull request for a new conflict

## Configuration

- **Repository**: Select the GitHub repository
- **Number**: Pull request number. Supports expressions.
- **Timeout Seconds**: Maximum time to poll before giving up. Default: 300.

## Output Channels

- **Clean**: GitHub reports ` + "`mergeable: true`" + `
- **Conflicted**: GitHub reports ` + "`mergeable: false`" + ` (this includes ` + "`mergeable_state: dirty`" + `)
- **Timed Out**: The timeout elapsed while ` + "`mergeable`" + ` was still ` + "`null`" + `

Each output includes the repository, number, head SHA, base ref, ` + "`mergeable`" + `, and ` + "`mergeableState`" + `.

A pull request blocked, behind, or unstable for reasons other than a conflict still reports ` + "`mergeable: true`" + ` and finishes on the Clean channel; this component only signals a real conflict.`
}

func (c *WaitForPullRequestMergeable) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "github.pullRequestMergeable",
		"data": map[string]any{
			"repository":     "acme/app",
			"number":         42,
			"sha":            "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
			"baseRef":        "main",
			"mergeable":      false,
			"mergeableState": "dirty",
		},
	}
}

func (c *WaitForPullRequestMergeable) Icon() string {
	return "github"
}

func (c *WaitForPullRequestMergeable) Color() string {
	return "gray"
}

func (c *WaitForPullRequestMergeable) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: waitMergeableCleanChannel, Label: "Clean"},
		{Name: waitMergeableConflictedChannel, Label: "Conflicted"},
		{Name: waitMergeableTimedOutChannel, Label: "Timed out"},
	}
}

func (c *WaitForPullRequestMergeable) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "number",
			Label:       "Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "42 or {{ root().data.pull_request.number }}",
			Description: "Pull request number. Supports expressions.",
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout Seconds",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     waitMergeableDefaultTimeoutSeconds,
			Description: "Maximum seconds to poll GitHub for a definite mergeable value.",
		},
	}
}

func (c *WaitForPullRequestMergeable) Setup(ctx core.SetupContext) error {
	if err := common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration); err != nil {
		return err
	}

	config, err := decodeWaitMergeableConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	if !common.IsExpression(pullNumberText(config.Number)) {
		if _, err := parsePullNumber(config.Number); err != nil {
			return err
		}
	}

	return nil
}

func (c *WaitForPullRequestMergeable) Execute(ctx core.ExecutionContext) error {
	config, err := decodeWaitMergeableConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	timeoutAt := time.Now().Add(config.timeout())
	if err := ctx.Metadata.Set(WaitForPullRequestMergeableMetadata{TimeoutAtUnix: timeoutAt.Unix()}); err != nil {
		return err
	}

	return evaluateWaitForPullRequestMergeable(waitMergeableRuntime{
		Configuration:  config,
		HTTP:           ctx.HTTP,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Integration:    ctx.Integration,
	})
}

func (c *WaitForPullRequestMergeable) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: waitMergeableEvaluateHook,
			Type: core.HookTypeInternal,
		},
	}
}

func (c *WaitForPullRequestMergeable) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != waitMergeableEvaluateHook {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	config, err := decodeWaitMergeableConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	return evaluateWaitForPullRequestMergeable(waitMergeableRuntime{
		Configuration:  config,
		HTTP:           ctx.HTTP,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Integration:    ctx.Integration,
	})
}

func (c *WaitForPullRequestMergeable) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *WaitForPullRequestMergeable) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *WaitForPullRequestMergeable) Cleanup(ctx core.SetupContext) error {
	return nil
}

type waitMergeableRuntime struct {
	Configuration  WaitForPullRequestMergeableConfiguration
	HTTP           core.HTTPContext
	Metadata       core.MetadataWriter
	ExecutionState core.ExecutionStateContext
	Requests       core.RequestContext
	Integration    core.IntegrationContext
}

func evaluateWaitForPullRequestMergeable(ctx waitMergeableRuntime) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata, err := decodeWaitMergeableMetadata(ctx.Metadata.Get())
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	number, err := parsePullNumber(ctx.Configuration.Number)
	if err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	pullRequest, _, err := client.GetPullRequest(context.Background(), ctx.Configuration.Repository, number)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", explainGitHubError(err))
	}

	output := WaitForPullRequestMergeableOutput{
		Repository:     ctx.Configuration.Repository,
		Number:         number,
		SHA:            pullRequest.GetHead().GetSHA(),
		BaseRef:        pullRequest.GetBase().GetRef(),
		Mergeable:      pullRequest.Mergeable,
		MergeableState: pullRequest.GetMergeableState(),
	}

	if pullRequest.Mergeable == nil {
		timeoutAt := time.Unix(metadata.TimeoutAtUnix, 0)
		now := time.Now()
		if !now.Before(timeoutAt) {
			return ctx.ExecutionState.Emit(waitMergeableTimedOutChannel, "github.pullRequestMergeable", []any{output})
		}

		delay := waitMergeablePollInterval
		if remaining := timeoutAt.Sub(now); remaining < delay {
			delay = remaining
		}
		if delay <= 0 {
			delay = time.Second
		}
		return ctx.Requests.ScheduleActionCall(waitMergeableEvaluateHook, map[string]any{}, delay)
	}

	channel := waitMergeableCleanChannel
	if !*pullRequest.Mergeable || pullRequest.GetMergeableState() == mergeableStateDirty {
		channel = waitMergeableConflictedChannel
	}

	return ctx.ExecutionState.Emit(channel, "github.pullRequestMergeable", []any{output})
}

func decodeWaitMergeableConfig(raw any) (WaitForPullRequestMergeableConfiguration, error) {
	var config WaitForPullRequestMergeableConfiguration
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &config,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	})
	if err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if config.Repository == "" {
		return config, fmt.Errorf("repository is required")
	}
	if pullNumberText(config.Number) == "" {
		return config, fmt.Errorf("number is required")
	}
	return config, nil
}

func decodeWaitMergeableMetadata(raw any) (WaitForPullRequestMergeableMetadata, error) {
	var metadata WaitForPullRequestMergeableMetadata
	if raw == nil {
		return metadata, fmt.Errorf("metadata is empty")
	}
	if err := mapstructure.Decode(raw, &metadata); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func (c WaitForPullRequestMergeableConfiguration) timeoutSeconds() int {
	if c.TimeoutSeconds == nil {
		return waitMergeableDefaultTimeoutSeconds
	}
	if *c.TimeoutSeconds <= 0 {
		return waitMergeableDefaultTimeoutSeconds
	}
	return *c.TimeoutSeconds
}

func (c WaitForPullRequestMergeableConfiguration) timeout() time.Duration {
	return time.Duration(c.timeoutSeconds()) * time.Second
}
