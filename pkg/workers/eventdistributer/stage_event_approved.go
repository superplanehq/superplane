package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

// HandleStageEventApproved processes a stage event approved message and forwards it to websocket clients
func HandleStageEventApproved(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received stage_event_approved event")

	// Parse the message as JSON
	var rawMsg map[string]interface{}
	if err := json.Unmarshal(messageBody, &rawMsg); err != nil {
		log.Warnf("Failed to unmarshal StageEventApproved message as JSON: %v, trying to continue", err)
		// If we can't parse it, create a minimal event
		rawMsg = map[string]interface{}{
			"event": "stage_event_approved",
		}
	}

	// Extract important fields
	eventID, _ := rawMsg["event_id"].(string)
	if eventID == "" {
		eventID, _ = rawMsg["id"].(string)
	}

	stageID, _ := rawMsg["stage_id"].(string)
	canvasID, _ := rawMsg["canvas_id"].(string)

	// Since we don't have access to the actual gRPC service anymore,
	// we'll just use the raw message data we received
	payload := map[string]interface{}{
		"id":        eventID,
		"stage_id":  stageID,
		"canvas_id": canvasID,
		"approved":  true,
	}

	// Copy any additional fields from the raw message
	for k, v := range rawMsg {
		// Don't overwrite our existing fields
		if _, exists := payload[k]; !exists {
			payload[k] = v
		}
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
