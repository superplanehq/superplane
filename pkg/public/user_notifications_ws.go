package public

import (
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/eventdistributer"
)

// handleUserNotificationsWebSocket authenticates an org member and
// subscribes the caller to live notifications for that user only.
func (s *Server) handleUserNotificationsWebSocket(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindUser,
			telemetry.WebSocketConnectionOutcomeAuthError,
		)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		telemetry.RecordWebSocketConnectionOutcome(
			r.Context(),
			ws.KindUser,
			telemetry.WebSocketConnectionOutcomeUpgradeError,
		)
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.WithError(err).Error("failed to upgrade user notifications websocket")
		}
		return
	}

	client := s.wsHub.NewClient(conn, eventdistributer.UserNotificationWebsocketTopic(user.ID.String()))
	<-client.Done
}
