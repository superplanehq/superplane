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
	resourceTypeDataSource   = "data-source"
	resourceTypeAlertRule    = "alert-rule"
	resourceTypeContactPoint = "contact-point"
	resourceTypeRuleGroup    = "rule-group"
	resourceTypeDashboard    = "dashboard"
	resourceTypePanel        = "panel"
	resourceTypeAnnotation   = "annotation"
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
	return "Connect Grafana alerts, alert rules, annotations, and data queries to SuperPlane workflows"
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
		&CreateAlertRule{},
		&DeleteAlertRule{},
		&GetAlertRule{},
		&ListAlertRules{},
		&QueryDataSource{},
		&UpdateAlertRule{},
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
	case resourceTypeFolder, resourceTypeDataSource, resourceTypeAlertRule, resourceTypeContactPoint, resourceTypeRuleGroup,
		resourceTypeDashboard, resourceTypePanel, resourceTypeAnnotation:
	default:
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	switch resourceType {
	case resourceTypeFolder:
		folders, err := client.ListFolders()
		if err != nil {
			return nil, err
		}
		return grafanaResourcesFromList(resourceTypeFolder, folders, func(f Folder) string { return f.UID }, func(f Folder) string { return f.Title }), nil
	case resourceTypeDataSource:
		dataSources, err := client.ListDataSources()
		if err != nil {
			return nil, err
		}
		return grafanaResourcesFromList(resourceTypeDataSource, dataSources, func(ds DataSource) string { return ds.UID }, func(ds DataSource) string { return ds.Name }), nil
	case resourceTypeAlertRule:
		alertRules, err := client.ListAlertRules("", "")
		if err != nil {
			return nil, err
		}
		return grafanaResourcesFromList(resourceTypeAlertRule, alertRules, func(r AlertRuleSummary) string { return r.UID }, func(r AlertRuleSummary) string { return r.Title }), nil
	case resourceTypeContactPoint:
		contactPoints, err := client.ListContactPoints()
		if err != nil {
			return nil, err
		}
		return grafanaResourcesFromList(resourceTypeContactPoint, contactPoints, func(cp ContactPoint) string { return cp.Name }, func(cp ContactPoint) string { return cp.Name }), nil
	case resourceTypeRuleGroup:
		groups, err := client.ListRuleGroups()
		if err != nil {
			return nil, err
		}
		resources := make([]core.IntegrationResource, 0, len(groups))
		for _, group := range groups {
			resources = append(resources, core.IntegrationResource{
				Type: resourceTypeRuleGroup,
				Name: group,
				ID:   group,
			})
		}
		return resources, nil
	case resourceTypeDashboard:
		dashboards, err := client.SearchDashboards()
		if err != nil {
			return nil, err
		}
		return grafanaResourcesFromList(resourceTypeDashboard, dashboards, func(d DashboardSearchHit) string { return d.UID }, func(d DashboardSearchHit) string { return d.Title }), nil
	case resourceTypePanel:
		dashboardUID := strings.TrimSpace(ctx.Parameters["dashboard"])
		if dashboardUID == "" {
			dashboardUID = strings.TrimSpace(ctx.Parameters["dashboardUID"])
		}
		if dashboardUID == "" {
			return []core.IntegrationResource{}, nil
		}

		panels, err := client.ListDashboardPanels(dashboardUID)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(panels))
		for _, panel := range panels {
			if panel.ID <= 0 {
				continue
			}

			name := strings.TrimSpace(panel.Title)
			if name == "" {
				name = fmt.Sprintf("Panel %d", panel.ID)
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceTypePanel,
				Name: name,
				ID:   strconv.FormatInt(panel.ID, 10),
			})
		}

		return resources, nil
	case resourceTypeAnnotation:
		annotations, err := client.ListAnnotations(nil, "", nil, 0, 0, 5000)
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(annotations))
		for _, annotation := range annotations {
			resources = append(resources, core.IntegrationResource{
				Type: resourceTypeAnnotation,
				Name: formatAnnotationResourceName(annotation),
				ID:   strconv.FormatInt(annotation.ID, 10),
			})
		}

		return resources, nil
	default:
		return nil, fmt.Errorf("internal error: unhandled grafana resource type %q", resourceType)
	}
}

func formatAnnotationResourceName(a Annotation) string {
	text := strings.TrimSpace(a.Text)
	const maxRunes = 72
	if text != "" {
		r := []rune(text)
		if len(r) > maxRunes {
			text = string(r[:maxRunes]) + "…"
		}
	}
	if text == "" {
		return fmt.Sprintf("#%d", a.ID)
	}
	return fmt.Sprintf("#%d · %s", a.ID, text)
}

func grafanaResourcesFromList[T any](resourceType string, items []T, idOf func(T) string, nameOf func(T) string) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(idOf(item))
		if id == "" {
			continue
		}

		name := strings.TrimSpace(nameOf(item))
		if name == "" {
			name = id
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   id,
		})
	}

	return resources
}
