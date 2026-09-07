package vercel

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("vercel", &Vercel{}, &WebhookHandler{})
}

type Vercel struct{}

type Configuration struct {
	AccessToken string `json:"accessToken" mapstructure:"accessToken"`
	TeamID      string `json:"teamId" mapstructure:"teamId"`
}

func (v *Vercel) Name() string {
	return "vercel"
}

func (v *Vercel) Label() string {
	return "Vercel"
}

func (v *Vercel) Icon() string {
	return "vercel"
}

func (v *Vercel) Description() string {
	return "Deploy to Vercel and react to deployment events"
}

func (v *Vercel) Instructions() string {
	return `
1. **Access Token:** Create it in [Vercel Account Settings -> Tokens](https://vercel.com/account/settings/tokens). Use a token scoped to the team that owns your projects.
2. **Team ID (optional):** Only needed when the token has access to multiple teams. Find it in [Team Settings](https://vercel.com/docs/accounts#find-your-team-id) (starts with ` + "`team_`" + `).
3. **Auth:** SuperPlane sends requests to the [Vercel REST API](https://vercel.com/docs/rest-api) using ` + "`Authorization: Bearer <ACCESS_TOKEN>`" + `.
4. **Git-connected projects only:** The Deploy component starts deployments for projects connected to GitHub, GitLab, or Bitbucket. Projects deployed through the CLI without a Git connection are not supported.
5. **Webhooks:** SuperPlane creates a Vercel webhook automatically via the [Webhooks API](https://vercel.com/docs/webhooks). No manual setup is required.`
}

func (v *Vercel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "accessToken",
			Label:       "Access Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Vercel access token",
		},
		{
			Name:        "teamId",
			Label:       "Team ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Vercel team ID. Use this if your access token has access to multiple teams.",
		},
	}
}

func (v *Vercel) Actions() []core.Action {
	return []core.Action{
		&TriggerDeployment{},
		&GetDeployment{},
		&ListDeployments{},
		&CancelDeployment{},
		&RollbackProduction{},
		&GetProject{},
		&CreateProject{},
		&UpsertEnvVar{},
		&AddDomain{},
		&RemoveDomain{},
	}
}

func (v *Vercel) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnDeployment{},
	}
}

func (v *Vercel) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (v *Vercel) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.AccessToken) == "" {
		return fmt.Errorf("accessToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if _, err := client.GetUser(); err != nil {
		return fmt.Errorf("failed to verify Vercel credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (v *Vercel) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (v *Vercel) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "project":
		return listProjects(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listProjects(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	// ponytail: first page of 100 projects only; add cursor pagination when someone hits it
	projects, err := client.ListProjects(100)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		if project.ID == "" || project.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: "project", Name: project.Name, ID: project.ID})
	}

	return resources, nil
}

func (v *Vercel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (v *Vercel) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
