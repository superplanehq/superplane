package pulls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const ThreadIDsExpression = `uniq(map(root().data.review_comments ?? [], .pull_request_review_thread?.id ?? ""))`

type ResolveReviewThread struct{}

type ResolveReviewThreadConfiguration struct {
	Repository          string `mapstructure:"repository" json:"repository"`
	ThreadID            string `mapstructure:"threadId" json:"threadId"`
	ThreadIDsExpression string `mapstructure:"threadIdsExpression" json:"threadIdsExpression"`
}

func (c *ResolveReviewThread) Name() string {
	return "github.resolveReviewThread"
}

func (c *ResolveReviewThread) Label() string {
	return "Resolve Review Thread"
}

func (c *ResolveReviewThread) Description() string {
	return "Mark one or more GitHub pull request review threads as resolved"
}

func (c *ResolveReviewThread) Documentation() string {
	return `The Resolve Review Thread component marks one or more GitHub pull request review threads as resolved, the same as clicking "Resolve conversation" in the GitHub UI.

## Use Cases

- **Automation feedback loops**: Resolve review threads after SuperPlane addresses inline review comments
- **Clean PR history**: Auto-resolve threads that have been addressed without manual intervention

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Thread ID**: A single GitHub review thread node ID (PRRT_...). Supports expressions.
- **Thread IDs Expression**: An expression that evaluates to a list of GitHub review thread node IDs (PRRT_...). Use this to resolve every thread of a review at once.

## Behavior

This component is idempotent: if a thread is already resolved, it succeeds without error.
If both Thread ID and Thread IDs Expression are set, Thread IDs Expression takes precedence.
Empty values are dropped, so a review that opens no inline threads finishes with success and emits no output.

The component resolves threads one by one. If one thread fails, the component resolves the
remaining threads and reports an error only when every thread failed.

## Permissions

The integration must have write access to pull requests. GitHub only exposes this operation through its GraphQL API.

## Output

Emits one object per resolved thread, including the thread id and its isResolved status.

## Example

Use this expression to resolve every thread of a submitted review:

    uniq(map(root().data.review_comments ?? [], .pull_request_review_thread?.id ?? ""))`
}

func (c *ResolveReviewThread) Icon() string {
	return "github"
}

func (c *ResolveReviewThread) Color() string {
	return "gray"
}

func (c *ResolveReviewThread) ExampleOutput() map[string]any {
	return map[string]any{
		"id":         "PRRT_kwDOABCD12MAAAABCDEFGH",
		"isResolved": true,
	}
}

func (c *ResolveReviewThread) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ResolveReviewThread) Configuration() []configuration.Field {
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
			Name:        "threadId",
			Label:       "Thread ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "PRRT_... or {{ previous().data.thread.id }}",
			Description: "GitHub review thread node ID (PRRT_...). Use for a single thread. If the resolved value is blank, the component finishes with success and emits no output.",
		},
		{
			Name:        "threadIdsExpression",
			Label:       "Thread IDs Expression",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Placeholder: ThreadIDsExpression,
			Description: "Expression that evaluates to a list of GitHub review thread node IDs (PRRT_...). Use this to resolve every thread of a review at once.",
		},
	}
}

func (c *ResolveReviewThread) Setup(ctx core.SetupContext) error {
	config, err := decodeResolveReviewThreadConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if !config.hasThreadInput() {
		return errors.New("thread ID or thread IDs expression is required")
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *ResolveReviewThread) Execute(ctx core.ExecutionContext) error {
	config, err := decodeResolveReviewThreadConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	threadIDs, err := config.threadIDs(ctx)
	if err != nil {
		return err
	}

	if len(threadIDs) == 0 {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"github.reviewThread",
			[]any{},
		)
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	resolved := make([]any, 0, len(threadIDs))
	errs := make([]error, 0, len(threadIDs))

	for _, threadID := range threadIDs {
		if err := client.ResolveReviewThread(context.Background(), threadID); err != nil {
			errs = append(errs, fmt.Errorf("failed to resolve thread %s: %w", threadID, err))
			continue
		}

		resolved = append(resolved, map[string]any{
			"id":         threadID,
			"isResolved": true,
		})
	}

	if len(errs) > 0 && len(resolved) == 0 {
		return fmt.Errorf("failed to resolve any review threads: %w", errors.Join(errs...))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.reviewThread",
		resolved,
	)
}

func (c *ResolveReviewThread) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *ResolveReviewThread) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ResolveReviewThread) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ResolveReviewThread) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ResolveReviewThread) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func decodeResolveReviewThreadConfiguration(raw any) (ResolveReviewThreadConfiguration, error) {
	var config ResolveReviewThreadConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return config, nil
}

func (c ResolveReviewThreadConfiguration) hasThreadInput() bool {
	if strings.TrimSpace(c.ThreadIDsExpression) != "" {
		return true
	}
	return strings.TrimSpace(c.ThreadID) != "" || common.IsExpression(c.ThreadID)
}

func (c ResolveReviewThreadConfiguration) threadIDs(ctx core.ExecutionContext) ([]string, error) {
	var candidates []string

	if strings.TrimSpace(c.ThreadIDsExpression) != "" {
		evaluated, err := ctx.Expressions.Run(c.ThreadIDsExpression)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate thread IDs expression: %w", err)
		}

		items, ok := evaluated.([]any)
		if !ok {
			return nil, fmt.Errorf("thread IDs expression must evaluate to a list, got %T", evaluated)
		}

		for _, item := range items {
			text, ok := item.(string)
			if !ok {
				continue
			}
			candidates = append(candidates, text)
		}
	} else {
		candidates = append(candidates, c.ThreadID)
	}

	return uniqueNonBlank(candidates), nil
}

func uniqueNonBlank(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, found := seen[trimmed]; found {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
