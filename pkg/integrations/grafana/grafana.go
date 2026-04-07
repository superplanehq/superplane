package grafana

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const resourceTypeDataSource = "data-source"
const resourceTypeSilence = "silence"

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

**Setup steps:**
1. In Grafana, go to **Administration → Users and access → Service Accounts**, select **Add service account**. 

   > **Service Account Role:**  
   > While naming the service account, go to **Roles → Basic roles** and select **Admin**.

	Navigate to the created service account and select **Add service account token**. Name it and set an expiration period then click **Generate token**. This is your **Service Account Token**.

2. Use your Grafana root URL as **Base URL** (for example ` + "`https://grafana.example.com`" + `).
3. Fill in **Base URL** and **Service Account Token** below, then save.
`
}

func (g *Grafana) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Description: "Your Grafana base URL (e.g. https://grafana.example.com or https://example.grafana.net)",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "Service Account Token",
			Type:        configuration.FieldTypeString,
			Description: "Grafana service account token with access to query data sources, unified alerting webhooks, and Alertmanager silences",
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
		&CreateSilence{},
		&DeleteSilence{},
		&GetSilence{},
		&ListSilences{},
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
	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	if resourceType == resourceTypeSilence {
		silences, err := client.ListSilences("")
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(silences))
		for _, silence := range silences {
			id := strings.TrimSpace(silence.ID)
			if id == "" {
				continue
			}

			label := formatSilenceResourceLabel(silence)
			if label == "" {
				label = id
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceTypeSilence,
				Name: label,
				ID:   id,
			})
		}

		return resources, nil
	}

	if resourceType != resourceTypeDataSource {
		return []core.IntegrationResource{}, nil
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

func formatSilenceResourceLabel(s Silence) string {
	comment := strings.TrimSpace(s.Comment)
	state := strings.TrimSpace(s.Status.State)

	id := strings.TrimSpace(s.ID)
	idShort := id
	if len(idShort) > 8 {
		idShort = idShort[:8]
	}

	if comment == "" && state == "" {
		return id
	}
	if comment == "" {
		return fmt.Sprintf("%s (%s)", idShort, state)
	}
	if state == "" {
		return fmt.Sprintf("%s (%s)", comment, idShort)
	}
	return fmt.Sprintf("%s [%s] (%s)", comment, state, idShort)
}
