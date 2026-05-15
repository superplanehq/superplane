package public

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

// handleAgentSessionWebSocket enforces ownership via the DB before
// subscribing — agent history is private to its creator.
func (s *Server) handleAgentSessionWebSocket(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(mux.Vars(r)["sessionId"])
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	enabled, err := models.HasExperimentalFeature(user.OrganizationID, features.FeatureClaudeManagedAgents)
	if err != nil {
		http.Error(w, "failed to load organization", http.StatusInternalServerError)
		return
	}
	if !enabled {
		http.Error(w, "agent chat is not enabled", http.StatusForbidden)
		return
	}

	if _, err := models.FindAgentSessionForUser(user.OrganizationID, user.ID, sessionID); err != nil {
		http.Error(w, "agent session not found", http.StatusNotFound)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.WithError(err).Error("failed to upgrade agent session websocket")
		}
		return
	}

	client := s.wsHub.NewClient(conn, eventdistributer.AgentSessionWebsocketTopic(sessionID.String()))
	<-client.Done
}
