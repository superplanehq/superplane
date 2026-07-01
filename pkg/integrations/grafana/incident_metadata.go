package grafana

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type IncidentNodeMetadata struct {
	Title    string `json:"title,omitempty" mapstructure:"title"`
	Status   string `json:"status,omitempty" mapstructure:"status"`
	Severity string `json:"severity,omitempty" mapstructure:"severity"`
	Label    string `json:"label,omitempty" mapstructure:"label"`
}

func resolveIncidentNodeMetadata(ctx core.SetupContext, incidentID string) error {
	if ctx.Metadata == nil {
		return nil
	}

	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" || isTemplateExpression(incidentID) {
		return ctx.Metadata.Set(IncidentNodeMetadata{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client during setup: %w", err)
	}

	incident, err := client.GetIncident(incidentID)
	if err != nil {
		return fmt.Errorf("error getting incident during setup: %w", err)
	}
	if incident == nil {
		return ctx.Metadata.Set(IncidentNodeMetadata{})
	}

	return ctx.Metadata.Set(IncidentNodeMetadata{
		Title:    strings.TrimSpace(incident.Title),
		Status:   strings.TrimSpace(incident.Status),
		Severity: strings.TrimSpace(incident.Severity),
		Label:    strings.TrimSpace(formatIncidentResourceLabel(incident.IncidentID, incident.Title, incident.Status)),
	})
}

func buildIncidentWebURL(ctx core.IntegrationContext, incidentID string) (string, error) {
	baseURL, err := readBaseURL(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s/a/grafana-irm-app/incidents/%s",
		strings.TrimSuffix(baseURL, "/"),
		url.PathEscape(strings.TrimSpace(incidentID)),
	), nil
}

func formatIncidentResourceLabel(id, title, status string) string {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	status = strings.TrimSpace(status)

	idShort := id
	if len(idShort) > 12 {
		idShort = idShort[:12]
	}

	if title == "" && status == "" {
		return id
	}
	if title == "" {
		return fmt.Sprintf("%s [%s]", idShort, status)
	}
	if status == "" {
		return fmt.Sprintf("%s (%s)", title, idShort)
	}
	return fmt.Sprintf("%s [%s] (%s)", title, status, idShort)
}
