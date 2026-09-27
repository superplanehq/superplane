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
	Repository string `mapstructure:"repository" json:"repository"`
	ThreadID   string `mapstructure:"threadId" json:"threadId"`
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
	return `The Resolve Review Thread component marks a GitHub pull request review thread as resolved, the same as clicking "Resolve conversation" in the GitHub UI.

## Use Cases

- **Automation feedback loops**: Resolve review threads after SuperPlane addresses inline review comments
- **Clean PR history**: Auto-resolve threads that have been addressed without manual intervention

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Thread ID**: The GitHub review thread node ID (PRRT_...). Supports expressions.

## Behavior

This component is idempotent: if the thread is already resolved, it succeeds without error.

## Permissions

The integration must have write access to pull requests. GitHub only exposes this operation through its GraphQL API.

## Output

Emits the thread object after resolution including id and isResolved status.`
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
			Required:    true,
			Placeholder: "PRRT_... or {{ previous().data.thread.id }}",
			Description: "GitHub review thread node ID (PRRT_...). If the resolved value is blank, the component finishes with success and emits no output.",
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

	if strings.TrimSpace(config.ThreadID) == "" && !common.IsExpression(config.ThreadID) {
		return errors.New("thread ID is required")
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

	threadID := strings.TrimSpace(config.ThreadID)
	if threadID == "" {
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

	err = client.ResolveReviewThread(context.Background(), threadID)
	if err != nil {
		return fmt.Errorf("failed to resolve review thread: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.reviewThread",
		[]any{map[string]any{
			"id":         threadID,
			"isResolved": true,
		}},
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
