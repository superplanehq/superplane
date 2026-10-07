package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

const (
	UserTopicPrefix       = "user:"
	UserNotificationEvent = "user_notification"
)

func UserNotificationWebsocketTopic(userID string) string {
	return UserTopicPrefix + userID
}

type userNotificationWebsocketEvent struct {
	Event   string                           `json:"event"`
	Payload messages.UserNotificationMessage `json:"payload"`
}

func HandleUserNotification(messageBody []byte, wsHub *ws.Hub) error {
	var msg messages.UserNotificationMessage
	if err := json.Unmarshal(messageBody, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal user notification: %w", err)
	}
	if msg.UserID == "" {
		return fmt.Errorf("missing userId in user notification")
	}

	payload, err := json.Marshal(userNotificationWebsocketEvent{
		Event:   UserNotificationEvent,
		Payload: msg,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal user notification websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(UserNotificationWebsocketTopic(msg.UserID), payload)
	log.Debugf("Broadcasted user_notification to user %s (order %s)", msg.UserID, msg.OrderID)
	return nil
}
