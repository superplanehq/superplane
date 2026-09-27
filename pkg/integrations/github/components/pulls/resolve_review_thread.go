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

type ResolveReviewThread struct{}

type ResolveReviewThreadConfiguration struct {
	Repository string   `mapstructure:"repository" json:"repository"`
	ThreadID   string   `mapstructure:"threadId" json:"threadId"`
	ThreadIDs  []string `mapstructure:"threadIds" json:"threadIds"`
}

func (c *ResolveReviewThread) Name() string {
	return "github.resolveReviewThread"
}

func (c *ResolveReviewThread) Label() string {
	return "Resolve Review Thread"
}

func (c *ResolveReviewThread) Description() string {
	return "Mark a GitHub pull request review thread as resolved"
}

func (c *ResolveReviewThread) Documentation() string {
	return `The Resolve Review Thread component marks one or more GitHub pull request review threads as resolved, the same as clicking "Resolve conversation" in the GitHub UI.

## Use Cases

- **Automation feedback loops**: Resolve review threads after SuperPlane addresses inline review comments
- **Clean PR history**: Auto-resolve threads that have been addressed without manual intervention

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Thread ID**: A single GitHub review thread node ID (PRRT_...). Supports expressions.
- **Thread IDs**: A list of GitHub review thread node IDs. Use this to resolve multiple threads at once.

## Behavior

This component is idempotent: if a thread is already resolved, it succeeds without error.
If both Thread ID and Thread IDs are provided, Thread IDs takes precedence.

## Permissions

The integration must have write access to pull requests. GitHub only exposes this operation through its GraphQL API.

## Output

Emits the thread object(s) after resolution including id and isResolved status.`
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
			Description: "GitHub review thread node ID (PRRT_...). Use for single thread. If the resolved value is blank, the component finishes with success and emits no output.",
		},
		{
			Name:        "threadIds",
			Label:       "Thread IDs",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "List of GitHub review thread node IDs (PRRT_...). Use to resolve multiple threads at once. Takes precedence over Thread ID.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Thread ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *ResolveReviewThread) Setup(ctx core.SetupContext) error {
	var config ResolveReviewThreadConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	hasThreadID := strings.TrimSpace(config.ThreadID) != "" || common.IsExpression(config.ThreadID)
	hasThreadIDs := len(config.ThreadIDs) > 0

	if !hasThreadID && !hasThreadIDs {
		return errors.New("thread ID or thread IDs is required")
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *ResolveReviewThread) Execute(ctx core.ExecutionContext) error {
	var config ResolveReviewThreadConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Repository) == "" {
		return errors.New("repository is required")
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	var threadIDs []string
	if len(config.ThreadIDs) > 0 {
		for _, id := range config.ThreadIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				threadIDs = append(threadIDs, trimmed)
			}
		}
	} else {
		threadID := strings.TrimSpace(config.ThreadID)
		if threadID != "" {
			threadIDs = append(threadIDs, threadID)
		}
	}

	if len(threadIDs) == 0 {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"github.reviewThread",
			[]any{},
		)
	}

	// Resolve all threads
	var resolvedThreads []map[string]any
	var errs []error

	for _, threadID := range threadIDs {
		err := client.ResolveReviewThread(context.Background(), threadID)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to resolve thread %s: %w", threadID, err))
			continue
		}
		resolvedThreads = append(resolvedThreads, map[string]any{
			"id":         threadID,
			"isResolved": true,
		})
	}

	if len(errs) > 0 && len(resolvedThreads) == 0 {
		return fmt.Errorf("failed to resolve any review threads: %v", errs)
	}

	result := make([]any, len(resolvedThreads))
	for i, t := range resolvedThreads {
		result[i] = t
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.reviewThread",
		result,
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
