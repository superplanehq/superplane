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
		dashboards, searchErr := client.SearchDashboards()
		if searchErr == nil {
			for _, dashboard := range dashboards {
				if strings.TrimSpace(dashboard.UID) != dashboardUID {
					continue
				}

				title := strings.TrimSpace(dashboard.Title)
				if title != "" {
					metadata.DashboardTitle = title
				}
				break
			}
		}
	}

	return ctx.Metadata.Set(metadata)
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
		annotations, listErr := client.ListAnnotations(nil, "", 0, 0, 5000)
		if listErr == nil {
			for _, annotation := range annotations {
				if strconv.FormatInt(annotation.ID, 10) != annotationID {
					continue
				}

				metadata.AnnotationLabel = formatAnnotationResourceName(annotation)
				break
			}
		}
	}

	return ctx.Metadata.Set(metadata)
}
