package bitbucket

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	AuthTypeAPIToken             = "apiToken"
	AuthTypeWorkspaceAccessToken = "workspaceAccessToken"

	installationInstructions = `
To configure Bitbucket with SuperPlane:

- **API Token mode**:
	- Go to **Atlassian Settings → Security → Create API token**.
	- Select **Bitbucket** App.
	- Create a token with admin:workspace:bitbucket scope.

- **Workspace Access Token mode**:
   - Go to **Bitbucket Workspace Settings → Security → Access tokens**.
   - Create a workspace access token.

- **Copy the token** and your workspace slug (for example: ` + "`my-workspace`" + `) below.
`
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("bitbucket", &Bitbucket{}, &BitbucketWebhookHandler{})
}

type Bitbucket struct{}

type Configuration struct {
	Workspace string  `json:"workspace"`
	AuthType  string  `json:"authType"`
	Token     *string `json:"token"`
	Email     *string `json:"email"`
}

type Metadata struct {
	AuthType  string             `json:"authType" mapstructure:"authType"`
	Workspace *WorkspaceMetadata `json:"workspace,omitempty" mapstructure:"workspace,omitempty"`
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
	return "React to events in your Bitbucket repositories"
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
			Required:    true,
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

func (b *Bitbucket) Components() []core.Component {
	return []core.Component{}
}

func (b *Bitbucket) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPush{},
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

	if config.Workspace == "" {
		return fmt.Errorf("workspace is required")
	}

	if config.AuthType == "" {
		return fmt.Errorf("authType is required")
	}

	if config.AuthType != AuthTypeAPIToken && config.AuthType != AuthTypeWorkspaceAccessToken {
		return fmt.Errorf("authType %s is not supported", config.AuthType)
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

func (b *Bitbucket) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (b *Bitbucket) Actions() []core.Action {
	return []core.Action{}
}

func (b *Bitbucket) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
