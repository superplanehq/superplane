package messages

import "encoding/json"

const (
	// UserNotificationRoutingKey: FactoryNotificationConsumer -> EventDistributer
	// -> the signed-in user's websocket.
	UserNotificationRoutingKey = "user-notification"
)

// UserNotificationMessage is a per-user live notification for one work
// order event. Field names match the camelCase websocket protocol.
type UserNotificationMessage struct {
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
	FactoryID      string `json:"factoryId"`
	FactoryKey     string `json:"factoryKey"`
	OrderID        string `json:"orderId"`
	OrderKey       string `json:"orderKey"`
	EventType      string `json:"eventType"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	URLPath        string `json:"urlPath"`
}

func (m UserNotificationMessage) Publish() error {
	return PublishUserNotification(m)
}

func PublishUserNotification(message UserNotificationMessage) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return Publish(CanvasExchange, UserNotificationRoutingKey, body)
}
