package public

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"gorm.io/gorm"
)

// adminResetOrganizationBacklogDefaults replaces every Backlog automation in
// the organization with the current SuperPlane defaults.
func (s *Server) adminResetOrganizationBacklogDefaults(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["orgId"]
	if _, err := uuid.Parse(orgID); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	webhookURL := strings.TrimSpace(s.WebhooksBaseURL)
	if webhookURL == "" {
		webhookURL = strings.TrimSpace(s.BaseURL)
	}

	result, err := factoryactions.ResetOrganizationBacklogDefaults(
		r.Context(),
		factoryactions.IntakeDependencies{
			Registry:       s.registry,
			Encryptor:      s.encryptor,
			AuthService:    s.authService,
			WebhookBaseURL: webhookURL,
		},
		orgID,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
			return
		}
		log.Errorf("admin: failed to reset Backlog defaults for org %s: %v", orgID, err)
		http.Error(w, "Failed to reset Backlog defaults", http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}
