package grafana

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("grafana", &Grafana{}, &GrafanaWebhookHandler{})
}

type Grafana struct{}

func (g *Grafana) Name() string {
	return "grafana"
}

func (g *Grafana) Label() string {
	return "Grafana"
}

func (g *Grafana) Icon() string {
	return "grafana"
}

func (g *Grafana) Description() string {
	return "Connect Grafana alerts and data queries to SuperPlane workflows"
}

func (g *Grafana) Instructions() string {
	return `
To connect Grafana:
1. Create a Service Account token or API key in Grafana (Configuration > API Keys or Service Accounts).
2. Set the Base URL to your Grafana instance (e.g. https://grafana.example.com).
3. Paste the API token into SuperPlane and save.

For the alert trigger:
1. SuperPlane will attempt to automatically create/update a Grafana Webhook contact point.
2. Route your alert rule to the contact point created by SuperPlane.
3. If auto-provisioning is not available (permissions/API limitations), create a Webhook contact point manually using the webhook URL from SuperPlane.
`
}

func (g *Grafana) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Description: "Your Grafana base URL (e.g. https://grafana.example.com)",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Description: "Grafana API key or service account token",
			Sensitive:   true,
			Required:    false,
		},
	}
}

func (g *Grafana) Actions() []core.Action {
	return []core.Action{}
}

func (g *Grafana) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (g *Grafana) Components() []core.Component {
	return []core.Component{
		&QueryDataSource{},
	}
}

func (g *Grafana) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertFiring{},
	}
}

func (g *Grafana) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (g *Grafana) Sync(ctx core.SyncContext) error {
	baseURL, err := ctx.Integration.GetConfig("baseURL")
	if err != nil {
		return fmt.Errorf("error reading baseURL: %v", err)
	}

	baseURLRaw := strings.TrimSpace(string(baseURL))
	if baseURL == nil || baseURLRaw == "" {
		return fmt.Errorf("baseURL is required")
	}

	parsed, err := url.Parse(baseURLRaw)
	if err != nil {
		return fmt.Errorf("invalid baseURL: %v", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid baseURL: must include scheme and host (e.g. https://grafana.example.com)")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid baseURL: unsupported scheme %q (expected http or https)", parsed.Scheme)
	}

	ctx.Integration.Ready()
	return nil
}

func (g *Grafana) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(404)
}

func (g *Grafana) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}
