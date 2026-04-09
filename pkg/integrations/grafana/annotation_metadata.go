package grafana

import (
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type AnnotationNodeMetadata struct {
	DashboardTitle  string `json:"dashboardTitle,omitempty" mapstructure:"dashboardTitle"`
	AnnotationLabel string `json:"annotationLabel,omitempty" mapstructure:"annotationLabel"`
}

func setDashboardNodeMetadata(ctx core.SetupContext, dashboardUID string) error {
	metadata := AnnotationNodeMetadata{}
	dashboardUID = strings.TrimSpace(dashboardUID)
	if dashboardUID == "" {
		return ctx.Metadata.Set(metadata)
	}

	if isExpressionValue(dashboardUID) {
		metadata.DashboardTitle = dashboardUID
		return ctx.Metadata.Set(metadata)
	}

	metadata.DashboardTitle = dashboardUID

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err == nil {
		if title, getErr := client.GetDashboardTitle(dashboardUID); getErr == nil && strings.TrimSpace(title) != "" {
			metadata.DashboardTitle = title
		} else if shouldFallbackDashboardTitleLookup(getErr) {
			if title := searchDashboardTitle(client, dashboardUID); title != "" {
				metadata.DashboardTitle = title
			}
		}
	}

	return ctx.Metadata.Set(metadata)
}

func shouldFallbackDashboardTitleLookup(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "response too large")
}

func searchDashboardTitle(client *Client, dashboardUID string) string {
	dashboards, err := client.SearchDashboards()
	if err != nil {
		return ""
	}

	for _, dashboard := range dashboards {
		if strings.TrimSpace(dashboard.UID) != dashboardUID {
			continue
		}

		return strings.TrimSpace(dashboard.Title)
	}

	return ""
}

func setAnnotationNodeMetadata(ctx core.SetupContext, annotationID string) error {
	metadata := AnnotationNodeMetadata{}
	annotationID = strings.TrimSpace(annotationID)
	if annotationID == "" {
		return ctx.Metadata.Set(metadata)
	}

	if isExpressionValue(annotationID) {
		metadata.AnnotationLabel = annotationID
		return ctx.Metadata.Set(metadata)
	}

	metadata.AnnotationLabel = annotationID

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err == nil {
		if id, parseErr := strconv.ParseInt(annotationID, 10, 64); parseErr == nil {
			if annotation, getErr := client.GetAnnotation(id); getErr == nil {
				metadata.AnnotationLabel = formatAnnotationResourceName(annotation)
			}
		}
	}

	return ctx.Metadata.Set(metadata)
}
