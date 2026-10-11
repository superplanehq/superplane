package bitbucket

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	AuthTypeAPIToken              = "apiToken"
	AuthTypeWorkspaceAccessToken  = "workspaceAccessToken"
	AuthTypeRepositoryAccessToken = "repositoryAccessToken"
	AuthTypeForgeApp              = "forgeApp"

	installationInstructions = `
To configure Bitbucket with SuperPlane:

- **API Token mode**:
	- Go to **Atlassian Settings → Security → Create API token**.
	- Select **Bitbucket** App.
	- Create a token with repository read/write and pull request read/write scopes, plus workspace read for repository listing.

- **Workspace Access Token mode**:
   - Go to **Bitbucket Workspace Settings → Security → Access tokens**.
   - Create a workspace access token with repository read/write, pull request read/write, and webhook read/write scopes.

- **Repository Access Token mode (single repository)**:
   - Go to **Repository Settings → Security → Access tokens**.
   - Create a repository access token with repository read/write, pull request read/write, and webhook read/write scopes.
   - This token works only for the configured repository and cannot list the workspace.

- **Copy the token** and your workspace slug (for example: ` + "`my-workspace`" + `) below.
For repository tokens, also copy the repository in workspace/repository format.
`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("bitbucket", &Bitbucket{}, &BitbucketWebhookHandler{})
}

type Bitbucket struct{}

type Configuration struct {
	Workspace  string  `json:"workspace"`
	Repository string  `json:"repository"`
	AuthType   string  `json:"authType"`
	Token      *string `json:"token"`
	Email      *string `json:"email"`
}

type Metadata struct {
	AuthType            string              `json:"authType" mapstructure:"authType"`
	Workspace           *WorkspaceMetadata  `json:"workspace,omitempty" mapstructure:"workspace,omitempty"`
	Repository          *RepositoryMetadata `json:"repository,omitempty" mapstructure:"repository,omitempty"`
	HostedApp           bool                `json:"hostedApp,omitempty" mapstructure:"hostedApp,omitempty"`
	ForgeInstallationID string              `json:"forgeInstallationId,omitempty" mapstructure:"forgeInstallationId,omitempty"`
}

type WorkspaceMetadata struct {
	UUID string `json:"uuid" mapstructure:"uuid"`
	Name string `json:"name" mapstructure:"name"`
	Slug string `json:"slug" mapstructure:"slug"`
}

func (b *Bitbucket) Name() string {
	return "bitbucket"
}

func (b *Bitbucket) Label() string {
	return "Bitbucket"
}

func (b *Bitbucket) Icon() string {
	return "bitbucket"
}

func (b *Bitbucket) Description() string {
	return "React to events and manage pull requests in your Bitbucket repositories"
}

func (b *Bitbucket) Instructions() string {
	return installationInstructions
}

func (b *Bitbucket) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "workspace",
			Label:       "Workspace",
			Type:        configuration.FieldTypeString,
			Description: "Bitbucket workspace slug",
			Placeholder: "e.g. my-workspace",
			Required:    false,
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Description: "Bitbucket repository in workspace/repository format (repository access tokens only)",
			Placeholder: "e.g. my-workspace/my-repo",
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeRepositoryAccessToken}},
			},
		},
		{
			Name:        "authType",
			Label:       "Authentication Type",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Bitbucket authentication type",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "API Token", Value: AuthTypeAPIToken},
						{Label: "Workspace Access Token", Value: AuthTypeWorkspaceAccessToken},
						{Label: "Repository Access Token", Value: AuthTypeRepositoryAccessToken},
					},
				},
			},
		},
		{
			Name:        "token",
			Label:       "Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "The API token or workspace access token to use for authentication",
			Required:    true,
		},
		{
			Name:        "email",
			Label:       "Email",
			Type:        configuration.FieldTypeString,
			Description: "Atlassian account email",
			Required:    true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "authType", Values: []string{AuthTypeAPIToken}},
			},
		},
	}
}

func (b *Bitbucket) Actions() []core.Action {
	return []core.Action{
		&FindPullRequest{},
		&CreatePullRequest{},
		&UpdatePullRequest{},
		&CreatePullRequestComment{},
		&WaitForBuilds{},
	}
}

func (b *Bitbucket) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPush{},
		&OnPullRequest{},
		&OnPullRequestComment{},
	}
}

func (b *Bitbucket) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (b *Bitbucket) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	if config.AuthType == AuthTypeForgeApp {
		return syncForgeApp(ctx)
	}

	if config.AuthType != AuthTypeAPIToken && config.AuthType != AuthTypeWorkspaceAccessToken && config.AuthType != AuthTypeRepositoryAccessToken {
		return fmt.Errorf("authType %s is not supported", config.AuthType)
	}

	if config.AuthType == AuthTypeRepositoryAccessToken {
		return syncRepositoryAccessToken(ctx, config)
	}

	if strings.TrimSpace(config.Workspace) == "" {
		return fmt.Errorf("workspace is required")
	}

	client, err := NewClient(config.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	workspace, err := client.GetWorkspace(config.Workspace)
	if err != nil {
		return fmt.Errorf("error getting workspace: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		AuthType: config.AuthType,
		Workspace: &WorkspaceMetadata{
			UUID: workspace.UUID,
			Name: workspace.Name,
			Slug: workspace.Slug,
		},
	})

	ctx.Integration.Ready()

	return nil
}

func syncRepositoryAccessToken(ctx core.SyncContext, config Configuration) error {
	repository := strings.TrimSpace(config.Repository)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}
	workspaceSlug, _, ok := strings.Cut(repository, "/")
	if !ok || strings.TrimSpace(workspaceSlug) == "" {
		return fmt.Errorf("repository must be in workspace/repository format: %q", repository)
	}

	client, err := NewClient(config.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	repo, err := client.GetRepository(repository)
	if err != nil {
		return fmt.Errorf("error getting repository: %w", err)
	}

	workspaceSlug = strings.TrimSpace(workspaceSlug)
	if repo.FullName != "" {
		if ws, _, ok := strings.Cut(strings.TrimSpace(repo.FullName), "/"); ok && strings.TrimSpace(ws) != "" {
			workspaceSlug = strings.TrimSpace(ws)
		}
	}
	workspaceName := workspaceSlug
	if repo.Workspace != nil && strings.TrimSpace(repo.Workspace.Slug) != "" {
		workspaceSlug = strings.TrimSpace(repo.Workspace.Slug)
		if strings.TrimSpace(repo.Workspace.Name) != "" {
			workspaceName = strings.TrimSpace(repo.Workspace.Name)
		}
	}

	ctx.Integration.SetMetadata(Metadata{
		AuthType: config.AuthType,
		Workspace: &WorkspaceMetadata{
			Slug: workspaceSlug,
			Name: workspaceName,
		},
		Repository: &RepositoryMetadata{
			UUID:     repo.UUID,
			Name:     repo.Name,
			FullName: repo.FullName,
			Slug:     repo.Slug,
		},
	})

	ctx.Integration.Ready()

	return nil
}

func syncForgeApp(ctx core.SyncContext) error {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if strings.TrimSpace(metadata.ForgeInstallationID) == "" {
		ctx.Integration.Error("Reconnect Bitbucket")
		return fmt.Errorf("Reconnect Bitbucket")
	}
	if _, _, err := bitbucketapp.CurrentSystemToken(metadata.ForgeInstallationID); err != nil {
		ctx.Integration.Error("Reconnect Bitbucket")
		return fmt.Errorf("Reconnect Bitbucket")
	}
	ctx.Integration.Ready()
	return nil
}

func (b *Bitbucket) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (b *Bitbucket) Hooks() []core.Hook {
	return []core.Hook{}
}

func (b *Bitbucket) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
