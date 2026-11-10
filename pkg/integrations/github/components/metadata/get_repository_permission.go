package metadata

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type GetRepositoryPermission struct{}

type GetRepositoryPermissionConfiguration struct {
	Repository       string `json:"repository" mapstructure:"repository"`
	Username         string `json:"username" mapstructure:"username"`
	FallbackUsername string `json:"fallbackUsername" mapstructure:"fallbackUsername"`
}

func (c *GetRepositoryPermission) Name() string {
	return "github.getRepositoryPermission"
}

func (c *GetRepositoryPermission) Label() string {
	return "Get Repository Permission"
}

func (c *GetRepositoryPermission) Description() string {
	return "Get the role and permission level for a user on a GitHub repository"
}

func (c *GetRepositoryPermission) Documentation() string {
	return `The Get Repository Permission component retrieves a user's effective permission level for a GitHub repository.

## Use Cases

- **Access checks**: Verify if a user has expected repository access
- **Automation gates**: Branch workflow behavior by repository permissions
- **Auditing**: Inspect repository roles in automated compliance checks
- **Triage routing**: Route incidents based on whether a user can push/administer
`
}

func (c *GetRepositoryPermission) Icon() string {
	return "github"
}

func (c *GetRepositoryPermission) Color() string {
	return "gray"
}

func (c *GetRepositoryPermission) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRepositoryPermission) Configuration() []configuration.Field {
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
			Name:     "username",
			Label:    "Username",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:        "fallbackUsername",
			Label:       "Fallback username",
			Type:        configuration.FieldTypeString,
			Description: "Check this user if the primary user does not have confirmed repository access.",
		},
	}
}

func (c *GetRepositoryPermission) Setup(ctx core.SetupContext) error {
	var config GetRepositoryPermissionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Username == "" {
		return errors.New("username is required")
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *GetRepositoryPermission) Execute(ctx core.ExecutionContext) error {
	var config GetRepositoryPermissionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	username := normalizePermissionUsername(config.Username)
	fallback := normalizePermissionUsername(config.FallbackUsername)
	candidates := []string{username}
	if fallback != "" && fallback != username {
		candidates = append(candidates, fallback)
	}
	var lookupErrors []error
	var permission *github.RepositoryPermissionLevel
	for _, candidate := range candidates {
		if candidate == "" {
			lookupErrors = append(lookupErrors, errors.New("username is required"))
			continue
		}
		result, _, err := client.GetRepositoryPermissionLevel(context.Background(), config.Repository, candidate)
		if err != nil {
			lookupErrors = append(lookupErrors, fmt.Errorf("failed to get repository permission level for %s: %w", candidate, err))
			continue
		}
		if result == nil {
			lookupErrors = append(lookupErrors, fmt.Errorf("missing repository permission for %s", candidate))
			continue
		}
		switch result.GetPermission() {
		case "read", "triage", "write", "maintain", "admin":
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "github.repositoryPermission", []any{result})
		case "none":
			permission = result
		default:
			lookupErrors = append(lookupErrors, fmt.Errorf("invalid repository permission for %s", candidate))
		}
	}
	if len(lookupErrors) > 0 {
		return errors.Join(lookupErrors...)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.repositoryPermission",
		[]any{permission},
	)
}

func normalizePermissionUsername(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}

func (c *GetRepositoryPermission) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetRepositoryPermission) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRepositoryPermission) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetRepositoryPermission) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetRepositoryPermission) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
