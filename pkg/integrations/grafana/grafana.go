package grafana

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const resourceTypeDataSource = "data-source"

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
1. In Grafana, go to Administration > Users and access > Service accounts.
2. Create a Service Account and assign a role (Viewer/Editor/Admin as needed).
3. Open the Service Account and create a token. Copy it immediately.
4. (Legacy Grafana) If Service Accounts are unavailable, use an API key.
5. Set the Base URL to your Grafana instance (e.g. https://grafana.example.com).
6. Paste the token into SuperPlane and save.

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
	if _, err := readBaseURL(ctx.Integration); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (g *Grafana) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(404)
}

func (g *Grafana) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != resourceTypeDataSource {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	dataSources, err := client.ListDataSources()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(dataSources))
	for _, source := range dataSources {
		id := strings.TrimSpace(source.UID)
		if id == "" {
			continue
		}

		name := strings.TrimSpace(source.Name)
		if name == "" {
			name = id
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceTypeDataSource,
			Name: name,
			ID:   id,
		})
	}

	return resources, nil
}
