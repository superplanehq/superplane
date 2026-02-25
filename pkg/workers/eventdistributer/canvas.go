package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

const (
	CanvasUpdatedEvent = "canvas_updated"
	CanvasDeletedEvent = "canvas_deleted"
)

type CanvasStatePayload struct {
	ID string `json:"id"`
}

type CanvasStateWebsocketEvent struct {
	Event   string             `json:"event"`
	Payload CanvasStatePayload `json:"payload"`
}

func HandleCanvasUpdated(messageBody []byte, wsHub *ws.Hub) error {
	return handleCanvasState(messageBody, wsHub, CanvasUpdatedEvent)
}

func HandleCanvasDeleted(messageBody []byte, wsHub *ws.Hub) error {
	return handleCanvasState(messageBody, wsHub, CanvasDeletedEvent)
}

func handleCanvasState(messageBody []byte, wsHub *ws.Hub, eventName string) error {
	pbMsg := &pb.CanvasMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal %s message: %w", eventName, err)
	}

	canvasID := pbMsg.CanvasId
	if canvasID == "" {
		return fmt.Errorf("missing canvas id in %s message", eventName)
	}

	wsEvent, err := json.Marshal(CanvasStateWebsocketEvent{
		Event: eventName,
		Payload: CanvasStatePayload{
			ID: canvasID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %s websocket event: %w", eventName, err)
	}

	wsHub.BroadcastToWorkflow(canvasID, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, canvasID)

	return nil
}
