package grafana

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type DashboardNodeMetadata struct {
	DashboardTitle string `json:"dashboardTitle,omitempty" mapstructure:"dashboardTitle"`
}

func storeDashboardNodeMetadata(ctx core.SetupContext, dashboardUID string) {
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

	_ = ctx.Metadata.Set(DashboardNodeMetadata{
		DashboardTitle: dashboard.Title,
	})
}
