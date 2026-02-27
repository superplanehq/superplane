package honeycomb

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("honeycomb", &Honeycomb{}, &HoneycombWebhookHandler{})
}

type Honeycomb struct{}

type Configuration struct {
	Site            string `json:"site" mapstructure:"site"`
	ManagementKey   string `json:"managementKey" mapstructure:"managementKey"`
	TeamSlug        string `json:"teamSlug" mapstructure:"teamSlug"`
	EnvironmentSlug string `json:"environmentSlug" mapstructure:"environmentSlug"`
}

func (h *Honeycomb) Name() string {
	return "honeycomb"
}

func (h *Honeycomb) Label() string {
	return "Honeycomb"
}

func (h *Honeycomb) Icon() string {
	return "honeycomb"
}

func (h *Honeycomb) Description() string {
	return "Monitor observability alerts and send events to Honeycomb datasets"
}

func (h *Honeycomb) Instructions() string {
	return `
Connect Honeycomb to SuperPlane using a Management Key.

**Required configuration:**
- **Site**: US (api.honeycomb.io) or EU (api.eu1.honeycomb.io) based on your account region.
- **Management Key**: Found in Honeycomb under Team Settings > API Keys. Must be in format <keyID>:<secret>.
- **Team Slug**: Your team identifier, visible in the Honeycomb URL: honeycomb.io/<team-slug>.
- **Environment Slug**: The environment containing your datasets (e.g. "production"). Found under Team Settings > Environments.

SuperPlane will automatically validate your credentials and manage all necessary Honeycomb resources — webhook recipients for triggers and ingest keys for actions — so no manual setup is required.
`
}

func (h *Honeycomb) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "site",
			Label:    "Honeycomb Site",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "api.honeycomb.io",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US (api.honeycomb.io)", Value: "api.honeycomb.io"},
						{Label: "EU (api.eu1.honeycomb.io)", Value: "api.eu1.honeycomb.io"},
					},
				},
			},
			Description: "Select the Honeycomb API host for your account region.",
		},
		{
			Name:        "managementKey",
			Label:       "Management Key",
			Type:        configuration.FieldTypeString,
			Description: "Honeycomb Management key in format <keyID>:<secret>.",
			Sensitive:   true,
			Required:    true,
		},
		{
			Name:        "teamSlug",
			Label:       "Team Slug",
			Type:        configuration.FieldTypeString,
			Description: "Your team identifier, visible in the Honeycomb URL: honeycomb.io/<team-slug>.",
			Required:    true,
		},
		{
			Name:        "environmentSlug",
			Label:       "Environment Slug",
			Type:        configuration.FieldTypeString,
			Description: "The environment containing your datasets (e.g. \"production\"). Found under Team Settings > Environments.",
			Required:    true,
		},
	}
}

func (h *Honeycomb) Components() []core.Component {
	return []core.Component{
		&CreateEvent{},
	}
}

func (h *Honeycomb) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertFired{},
	}
}

func (h *Honeycomb) Actions() []core.Action {
	return []core.Action{}
}

func (h *Honeycomb) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (h *Honeycomb) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Honeycomb) Sync(ctx core.SyncContext) error {
	cfg := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(cfg.Site) == "" {
		return fmt.Errorf("site is required")
	}

	if strings.TrimSpace(cfg.ManagementKey) == "" {
		return fmt.Errorf("managementKey is required")
	}

	if strings.TrimSpace(cfg.TeamSlug) == "" {
		return fmt.Errorf("teamSlug is required")
	}

	if strings.TrimSpace(cfg.EnvironmentSlug) == "" {
		return fmt.Errorf("environmentSlug is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.ValidateManagementKey(cfg.TeamSlug); err != nil {
		return err
	}

	if err := client.EnsureConfigurationKey(cfg.TeamSlug); err != nil {
		return err
	}

	if err := client.EnsureIngestKey(cfg.TeamSlug); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (h *Honeycomb) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(404)
	_, _ = ctx.Response.Write([]byte("not found"))
}
