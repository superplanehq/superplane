package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

// HandleExecutionFinished processes an execution finished message and forwards it to websocket clients
func HandleExecutionFinished(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_finished event")

	// Parse the message as JSON
	var rawMsg map[string]interface{}
	if err := json.Unmarshal(messageBody, &rawMsg); err != nil {
		log.Warnf("Failed to unmarshal ExecutionFinished message as JSON: %v, trying to continue", err)
		// If we can't parse it, create a minimal event
		rawMsg = map[string]interface{}{
			"event": "execution_finished",
		}
	}

	// Extract canvas ID for routing
	canvasID, ok := rawMsg["canvas_id"].(string)
	if !ok {
		canvasID = ""
	}

	// Create the websocket event
	payload := map[string]interface{}{
		"id":        rawMsg["id"],
		"stage_id":  rawMsg["stage_id"],
		"canvas_id": canvasID,
		"result":    rawMsg["result"],
		"timestamp": rawMsg["timestamp"],
	}
	wsEvent := map[string]interface{}{
		"event":    "execution_finished",
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
		log.Debugf("Broadcasted execution_finished event to canvas %s", canvasID)
	} else {
		// Fall back to broadcasting to all clients
		wsHub.BroadcastAll(wsEventJSON)
		log.Debugf("Broadcasted execution_finished event to all clients")
	}

	return nil
}
