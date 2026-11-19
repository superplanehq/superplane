package public

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

func (s *Server) handleWebhookSetupWebSocket(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// The organization-wide stream is for browser sessions. Scoped API tokens
	// can have narrower access than the integration and factory lists.
	if _, scoped := middleware.GetScopedTokenClaimsFromContext(r.Context()); scoped {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	for _, resource := range []string{"integrations", "factories"} {
		allowed, err := s.authService.CheckOrganizationPermission(r.Context(), user.ID.String(), user.OrganizationID.String(), resource, "read")
		if err != nil {
			log.WithError(err).Error("Could not authorize webhook setup subscription")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !allowed {
			continue
		}
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := s.wsHub.NewClient(conn, ws.WebhookSetupTopic(user.OrganizationID))
		<-client.Done
		return
	}
	http.Error(w, "Forbidden", http.StatusForbidden)
}
