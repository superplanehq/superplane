package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type CanvasEventWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleCanvasEventCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_created event")

	pbMsg := &pb.CanvasNodeEventMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal execution_created message: %w", err)
	}

	return handleWorkflowEventState(pbMsg.CanvasId, pbMsg.Id, wsHub, "event_created")
}

func handleWorkflowEventState(canvasID string, eventID string, wsHub *ws.Hub, eventName string) error {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return fmt.Errorf("failed to parse canvas id: %w", err)
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("failed to parse event id: %w", err)
	}

	event, err := models.FindCanvasEventForCanvas(canvasUUID, eventUUID)
	if err != nil {
		return fmt.Errorf("failed to find event: %w", err)
	}

	serializedEvent, err := canvases.SerializeCanvasEvent(*event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	serializedEventJSON, err := protojson.Marshal(serializedEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	wsEvent, err := json.Marshal(CanvasEventWebsocketEvent{
		Event:   eventName,
		Payload: json.RawMessage(serializedEventJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(canvasID, wsEvent)
	log.Debugf("Broadcasted %s event to canvas %s", eventName, canvasID)

	return nil
}
