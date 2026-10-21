package public

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
)

func (s *Server) adminResetBuiltInAutomations(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	result, err := factoryactions.ResetOrganizationBuiltInAutomations(r.Context(), orgID, factoryactions.IntakeDependencies{
		Registry:       s.registry,
		Encryptor:      s.encryptor,
		AuthService:    s.authService,
		WebhookBaseURL: s.WebhooksBaseURL,
	})
	if err != nil {
		log.Errorf("admin: failed to reset built-in automations for org %s: %v", orgID, err)
		var settingsError *factoryactions.BuiltInAutomationSettingsError
		if errors.As(err, &settingsError) {
			http.Error(w, fmt.Sprintf("Reset stopped: %v. No automations were changed. Correct these settings and try again.", err), http.StatusBadRequest)
			return
		}
		http.Error(w, "Could not reset built-in automations.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Errorf("admin: failed to encode built-in automation reset: %v", err)
	}
}
