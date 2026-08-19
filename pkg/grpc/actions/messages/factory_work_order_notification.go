package messages

import "encoding/json"

// FactoryWorkOrderNotificationRoutingKey: work order activity -> FactoryNotificationConsumer -> notification emails.
const FactoryWorkOrderNotificationRoutingKey = "factory-work-order-notification"

// FactoryWorkOrderNotificationMessage carries the notification payload for
// one work order event. Publishers emit it after the database transaction
// commits, so the consumer never sees rolled-back activity. The payload is
// self-contained (actor, comment body, transition) to avoid re-reading the
// timeline and racing with later writes.
type FactoryWorkOrderNotificationMessage struct {
	OrganizationID string `json:"organization_id"`
	FactoryID      string `json:"factory_id"`
	OrderID        string `json:"order_id"`
	// EventType is one of the factory event type constants
	// (`order.comment.added`, `order.assignees.updated`, ...).
	EventType string `json:"event_type"`
	// ActorUserID identifies the user who caused the event. Empty for
	// automation actors; those recipients are never excluded.
	ActorUserID string `json:"actor_user_id,omitempty"`
	// ActorName is a display name for automation actors (app or node name).
	ActorName        string   `json:"actor_name,omitempty"`
	AssignedUserIDs  []string `json:"assigned_user_ids,omitempty"`
	CommentBody      string   `json:"comment_body,omitempty"`
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty"`
	FromState        string   `json:"from_state,omitempty"`
	ToState          string   `json:"to_state,omitempty"`
	Result           string   `json:"result,omitempty"`
	ArtifactType     string   `json:"artifact_type,omitempty"`
}

func (m FactoryWorkOrderNotificationMessage) Publish() error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return Publish(CanvasExchange, FactoryWorkOrderNotificationRoutingKey, body)
}
