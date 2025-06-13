package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

// HandleStageEventApproved processes a stage event approved message and forwards it to websocket clients
func HandleStageEventApproved(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_event_approved event")

	// Parse the message as JSON
	var rawMsg pb.StageEventApproved
	if err := proto.Unmarshal(messageBody, &rawMsg); err != nil {
		log.Warnf("Failed to unmarshal StageEventApproved message as JSON: %v, trying to continue", err)
		// If we can't parse it, create a minimal event
		rawMsg = pb.StageEventApproved{
			EventId:  "",
			StageId:  "",
			CanvasId: "",
			SourceId: "",
		}
	}

	// Extract important fields
	eventID := rawMsg.EventId
	stageID := rawMsg.StageId
	canvasID := rawMsg.CanvasId
	sourceID := rawMsg.SourceId

	// Since we don't have access to the actual gRPC service anymore,
	// we'll just use the raw message data we received
	payload := map[string]interface{}{
		"id":        eventID,
		"stage_id":  stageID,
		"canvas_id": canvasID,
		"source_id": sourceID,
		"approved":  true,
	}

	// Create the websocket event
	wsEvent := map[string]interface{}{
		"event":   "stage_event_approved",
		"payload": payload,
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to clients
	if canvasID != "" {
		// Send to specific canvas
		wsHub.BroadcastToCanvas(canvasID, wsEventJSON)
		log.Debugf("Broadcasted stage_event_approved event to canvas %s", canvasID)
	} else {
		// Fall back to broadcasting to all clients
		wsHub.BroadcastAll(wsEventJSON)
		log.Debugf("Broadcasted stage_event_approved event to all clients")
	}

	return nil
}
