package public

import (
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
)

// HandleLinearOAuthCallback finishes SuperPlane's hosted Linear OAuth app.
// Linear sends every grant to this one callback. The CSRF state finds the
// pending SuperPlane connection and is consumed on this first attempt.
func (s *Server) HandleLinearOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	integration, err := models.ClaimHostedLinearOAuthState(database.DB(r.Context()), state)
	if err != nil || !isHostedLinearOAuth(integration) {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	s.dispatchIntegrationRequest(w, r, integration)
}

func isHostedLinearOAuth(integration *models.Integration) bool {
	if integration == nil || integration.AppName != "linear" {
		return false
	}

	var metadata linear.Metadata
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return false
	}

	return metadata.HostedOAuth
}
