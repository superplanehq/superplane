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
	CanvasVersionDeletedEvent = "canvas_version_deleted"
	CanvasStagingUpdatedEvent = "staging_updated"
	CanvasDeletedEvent        = "canvas_deleted"
	CanvasMemoryUpdatedEvent  = "memory_updated"
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

func HandleCanvasMemoryUpdated(messageBody []byte, wsHub *ws.Hub) error {
	return handleCanvasState(messageBody, wsHub, CanvasMemoryUpdatedEvent)
}

func HandleCanvasVersionUpdated(messageBody []byte, wsHub *ws.Hub) error {
	return handleCanvasVersion(messageBody, wsHub, CanvasVersionUpdatedEvent)
}

func HandleCanvasVersionDeleted(messageBody []byte, wsHub *ws.Hub) error {
	return handleCanvasVersion(messageBody, wsHub, CanvasVersionDeletedEvent)
}

func HandleCanvasStagingUpdated(messageBody []byte, wsHub *ws.Hub) error {
	return handleCanvasVersion(messageBody, wsHub, CanvasStagingUpdatedEvent)
}

func handleCanvasVersion(messageBody []byte, wsHub *ws.Hub, eventName string) error {
	pbMsg := &pb.CanvasVersionMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal %s message: %w", eventName, err)
	}

	if pbMsg.CanvasId == "" {
		return fmt.Errorf("missing canvas id in %s message", eventName)
	}

	if pbMsg.VersionId == "" {
		return fmt.Errorf("missing version id in %s message", eventName)
	}

	wsEvent, err := json.Marshal(CanvasStateWebsocketEvent{
		Event: eventName,
		Payload: CanvasStatePayload{
			ID:        pbMsg.CanvasId,
			CanvasID:  pbMsg.CanvasId,
			VersionID: pbMsg.VersionId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %s websocket event: %w", eventName, err)
	}

	wsHub.BroadcastToWorkflow(pbMsg.CanvasId, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, pbMsg.CanvasId)

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
