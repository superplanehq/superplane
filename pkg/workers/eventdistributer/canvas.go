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
	CanvasUpdatedEvent        = "canvas_updated"
	CanvasVersionUpdatedEvent = "canvas_version_updated"
	CanvasDeletedEvent        = "canvas_deleted"
)

type CanvasStatePayload struct {
	ID        string `json:"id"`
	CanvasID  string `json:"canvasId,omitempty"`
	VersionID string `json:"versionId,omitempty"`
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

func HandleCanvasVersionUpdated(messageBody []byte, wsHub *ws.Hub) error {
	pbMsg := &pb.CanvasVersionMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal %s message: %w", CanvasVersionUpdatedEvent, err)
	}

	if pbMsg.CanvasId == "" {
		return fmt.Errorf("missing canvas id in %s message", CanvasVersionUpdatedEvent)
	}

	if pbMsg.VersionId == "" {
		return fmt.Errorf("missing version id in %s message", CanvasVersionUpdatedEvent)
	}

	wsEvent, err := json.Marshal(CanvasStateWebsocketEvent{
		Event: CanvasVersionUpdatedEvent,
		Payload: CanvasStatePayload{
			ID:        pbMsg.CanvasId,
			CanvasID:  pbMsg.CanvasId,
			VersionID: pbMsg.VersionId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %s websocket event: %w", CanvasVersionUpdatedEvent, err)
	}

	wsHub.BroadcastToWorkflow(pbMsg.CanvasId, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", CanvasVersionUpdatedEvent, pbMsg.CanvasId)

	return nil
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
			ID:       canvasID,
			CanvasID: canvasID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %s websocket event: %w", eventName, err)
	}

	wsHub.BroadcastToWorkflow(canvasID, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, canvasID)

	return nil
}
