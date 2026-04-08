package grafana

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type DashboardNodeMetadata struct {
	DashboardTitle string `json:"dashboardTitle,omitempty" mapstructure:"dashboardTitle"`
	PanelTitle     string `json:"panelTitle,omitempty" mapstructure:"panelTitle"`
	PanelLabel     string `json:"panelLabel,omitempty" mapstructure:"panelLabel"`
}

func storeDashboardNodeMetadata(ctx core.SetupContext, dashboardUID string, panelID *int) {
	trimmed := strings.TrimSpace(dashboardUID)
	if ctx.Metadata == nil || ctx.HTTP == nil || trimmed == "" {
		return
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return
	}

	dashboard, err := client.GetDashboard(trimmed)
	if err != nil || dashboard.Title == "" {
		return
	}

	metadata := DashboardNodeMetadata{
		DashboardTitle: dashboard.Title,
	}

	if panelID != nil {
		for _, panel := range dashboard.Panels {
			if panel.ID != *panelID {
				continue
			}

			metadata.PanelTitle = strings.TrimSpace(panel.Title)
			metadata.PanelLabel = strings.TrimSpace(formatPanelResourceLabel(panel))
			break
		}
	}

	_ = ctx.Metadata.Set(DashboardNodeMetadata{
		DashboardTitle: metadata.DashboardTitle,
		PanelTitle:     metadata.PanelTitle,
		PanelLabel:     metadata.PanelLabel,
	})
}
