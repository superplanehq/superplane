package public

import (
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/models"
)

// HandleJiraOAuthCallback finishes SuperPlane's hosted Atlassian OAuth app.
// Atlassian sends every grant to this one callback. The CSRF state finds
// the pending SuperPlane connection.
func (s *Server) HandleJiraOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	integration, err := models.FindJiraIntegrationByOAuthState(database.DB(r.Context()), state)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if !isHostedJiraOAuth(integration) {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	s.dispatchIntegrationRequest(w, r, integration)
}

func isHostedJiraOAuth(integration *models.Integration) bool {
	if integration == nil || integration.AppName != "jira" {
		return false
	}

	var metadata jira.Metadata
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return false
	}

	return metadata.HostedOAuth
}
