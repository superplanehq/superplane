package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("launchdarkly", &LaunchDarkly{})
}

type LaunchDarkly struct{}

type Configuration struct {
	APIAccessToken string `json:"apiAccessToken" mapstructure:"apiAccessToken"`
	APIBaseURL     string `json:"apiBaseUrl" mapstructure:"apiBaseUrl"`
}

func (l *LaunchDarkly) Name() string {
	return "launchdarkly"
}

func (l *LaunchDarkly) Label() string {
	return "LaunchDarkly"
}

func (l *LaunchDarkly) Icon() string {
	return "flag"
}

func (l *LaunchDarkly) Description() string {
	return "Manage and react to feature-flag changes in LaunchDarkly"
}

func (l *LaunchDarkly) Instructions() string {
	return `## Create a LaunchDarkly API token

1. Open [LaunchDarkly Authorization settings](https://app.launchdarkly.com/settings/authorization)
2. Create a token for SuperPlane
3. Grant permissions needed for your actions:
   - Read flags (for **Get Feature Flag**)
   - Delete/write flags (for **Delete Feature Flag**)
4. Copy the token and paste it in **API Access Token**

### EU tenant
If your account uses the EU region, set **API Base URL** to:

- https://app.eu.launchdarkly.com

If empty, SuperPlane uses the US default endpoint.`
}

func (l *LaunchDarkly) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiAccessToken",
			Label:       "API Access Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "LaunchDarkly API access token (personal or service token)",
		},
		{
			Name:        "apiBaseUrl",
			Label:       "API Base URL (Optional)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Regional LaunchDarkly host, e.g. https://app.eu.launchdarkly.com",
			Placeholder: "https://app.eu.launchdarkly.com",
		},
	}
}

func (l *LaunchDarkly) Components() []core.Component {
	return []core.Component{
		&GetFlag{},
		&DeleteFlag{},
	}
}

func (l *LaunchDarkly) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (l *LaunchDarkly) Sync(ctx core.SyncContext) error {
	cfg := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(cfg.APIAccessToken) == "" {
		return fmt.Errorf("apiAccessToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	if err := client.VerifyCredentials(); err != nil {
		return fmt.Errorf("failed to verify LaunchDarkly credentials: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (l *LaunchDarkly) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (l *LaunchDarkly) Actions() []core.Action {
	return []core.Action{}
}

func (l *LaunchDarkly) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (l *LaunchDarkly) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (l *LaunchDarkly) HandleRequest(ctx core.HTTPRequestContext) {}
