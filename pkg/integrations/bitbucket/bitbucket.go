package bitbucket

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("bitbucket", &Bitbucket{}, &BitbucketWebhookHandler{})
}

type Bitbucket struct{}

type Configuration struct {
	Workspace string `json:"workspace"`
	Email     string `json:"email"`
	APIToken  string `json:"apiToken"`
}

type Metadata struct {
	Workspace    string       `json:"workspace" mapstructure:"workspace"`
	Repositories []Repository `json:"repositories" mapstructure:"repositories"`
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
	return ""
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
			Name:        "email",
			Label:       "Email",
			Type:        configuration.FieldTypeString,
			Description: "Atlassian account email",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Bitbucket API token with repository read scope",
			Required:    true,
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	//
	// Validate credentials by listing repositories.
	//
	repos, err := client.ListRepositories(config.Workspace)
	if err != nil {
		return fmt.Errorf("error listing repositories: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		Workspace:    config.Workspace,
		Repositories: repos,
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
