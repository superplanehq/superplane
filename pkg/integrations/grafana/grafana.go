package grafana

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	resourceTypeDataSource = "data-source"
	resourceTypeDashboard  = "dashboard"
	resourceTypeAnnotation = "annotation"
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
			Description: "Grafana service account token with access to query data sources and manage alerting webhooks",
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
		&CreateAnnotation{},
		&ListAnnotations{},
		&DeleteAnnotation{},
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
	switch resourceType {
	case resourceTypeDataSource:
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

	case resourceTypeDashboard:
		client, err := NewClient(ctx.HTTP, ctx.Integration, true)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %w", err)
		}

		dashboards, err := client.SearchDashboards()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(dashboards))
		for _, d := range dashboards {
			id := strings.TrimSpace(d.UID)
			if id == "" {
				continue
			}

			name := strings.TrimSpace(d.Title)
			if name == "" {
				name = id
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceTypeDashboard,
				Name: name,
				ID:   id,
			})
		}

		return resources, nil

	case resourceTypeAnnotation:
		client, err := NewClient(ctx.HTTP, ctx.Integration, true)
		if err != nil {
			return nil, fmt.Errorf("error creating client: %w", err)
		}

		annotations, err := client.ListAnnotations(nil, "", 0, 0, 5000)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(annotations))
		for _, a := range annotations {
			idStr := strconv.FormatInt(a.ID, 10)
			name := formatAnnotationResourceName(a)
			resources = append(resources, core.IntegrationResource{
				Type: resourceTypeAnnotation,
				Name: name,
				ID:   idStr,
			})
		}

		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func formatAnnotationResourceName(a Annotation) string {
	text := strings.TrimSpace(a.Text)
	const maxLen = 72
	if len(text) > maxLen {
		text = text[:maxLen] + "…"
	}
	if text == "" {
		return fmt.Sprintf("#%d", a.ID)
	}
	return fmt.Sprintf("#%d · %s", a.ID, text)
}
