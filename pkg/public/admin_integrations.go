package public

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type adminIntegration struct {
	ID               string            `json:"id"`
	AppName          string            `json:"app_name"`
	InstallationName string            `json:"installation_name"`
	State            string            `json:"state"`
	StateDescription string            `json:"state_description"`
	Details          map[string]string `json:"details"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

// adminListOrgIntegrations returns the connections of an organization with
// their state. Metadata can hold CSRF state and other private values, so only
// allowlisted identifiers leave the server.
func (s *Server) adminListOrgIntegrations(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]

	org, err := models.FindOrganizationByID(orgID)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	search, limit, offset := parsePagination(r)
	integrations, total, err := models.ListIntegrationsPage(database.DB(r.Context()), org.ID, search, limit, offset)
	if err != nil {
		log.Errorf("admin: failed to list integrations for org %s: %v", orgID, err)
		http.Error(w, "Failed to list connections", http.StatusInternalServerError)
		return
	}

	items := make([]adminIntegration, 0, len(integrations))
	for _, integration := range integrations {
		items = append(items, adminIntegration{
			ID:               integration.ID.String(),
			AppName:          integration.AppName,
			InstallationName: integration.InstallationName,
			State:            integration.State,
			StateDescription: integration.StateDescription,
			Details:          adminIntegrationDetails(integration.Metadata.Data()),
			CreatedAt:        formatAdminTime(integration.CreatedAt),
			UpdatedAt:        formatAdminTime(integration.UpdatedAt),
		})
	}

	respondJSON(w, paginatedResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func adminIntegrationDetails(metadata map[string]any) map[string]string {
	details := map[string]string{}
	addAdminDetail(details, "installation_uuid", metadata["installationUUID"])
	addAdminDetail(details, "installation_id", metadata["installationId"])
	addAdminDetail(details, "hosted_app", metadata["hostedApp"])
	if hosted, ok := metadata["hostedOAuth"].(bool); ok && hosted {
		addAdminDetail(details, "hosted_app", true)
	}
	if organization, ok := metadata["organization"].(map[string]any); ok {
		addAdminDetail(details, "external_organization", organization["slug"])
	}
	if name, ok := metadata["organization"].(string); ok {
		addAdminDetail(details, "external_organization", name)
	}
	addAdminDetail(details, "workspace_key", metadata["urlKey"])
	return details
}

func addAdminDetail(details map[string]string, key string, value any) {
	if value == nil {
		return
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "false" {
		return
	}
	details[key] = text
}

func formatAdminTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
