package public

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

// handleFactoryWebSocket authenticates an org member and verifies the factory
// belongs to that organization before subscribing to work-order updates.
func (s *Server) handleFactoryWebSocket(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindFactory,
			telemetry.WebSocketConnectionOutcomeAuthError,
		)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	factoryID, err := uuid.Parse(mux.Vars(r)["factoryId"])
	if err != nil {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindFactory,
			telemetry.WebSocketConnectionOutcomeAuthError,
		)
		http.Error(w, "invalid factory id", http.StatusBadRequest)
		return
	}

	enabled, err := models.HasExperimentalFeature(user.OrganizationID, features.FeatureFactories)
	if err != nil {
		http.Error(w, "failed to load organization", http.StatusInternalServerError)
		return
	}
	if !enabled {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindFactory,
			telemetry.WebSocketConnectionOutcomeAuthError,
		)
		http.Error(w, "factories are not enabled", http.StatusForbidden)
		return
	}

	if _, err := models.FindFactory(database.DB(r.Context()), user.OrganizationID, factoryID); err != nil {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindFactory,
			telemetry.WebSocketConnectionOutcomeAuthError,
		)
		http.Error(w, "factory not found", http.StatusNotFound)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindFactory,
			telemetry.WebSocketConnectionOutcomeUpgradeError,
		)
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.WithError(err).Error("failed to upgrade factory websocket")
		}
		return
	}

	client := s.wsHub.NewClient(conn, eventdistributer.FactoryWebsocketTopic(factoryID.String()))
	<-client.Done
}
