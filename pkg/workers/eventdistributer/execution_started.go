package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

// HandleExecutionStarted processes an execution started message and forwards it to websocket clients
func HandleExecutionStarted(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_started event")

	// Parse the message as JSON (we expect it to be a JSON-encoded message)
	var rawMsg map[string]interface{}
	if err := json.Unmarshal(messageBody, &rawMsg); err != nil {
		log.Warnf("Failed to unmarshal ExecutionStarted message as JSON: %v, trying to continue", err)
		// If we can't parse it, create a minimal event
		rawMsg = map[string]interface{}{
			"event": "execution_started",
		}
	}
	
	// Get execution ID if available
	executionIDStr, _ := rawMsg["execution_id"].(string)
	if executionIDStr == "" {
		executionIDStr, _ = rawMsg["id"].(string)
	}
	
	// Get canvas ID if available
	canvasID, _ := rawMsg["canvas_id"].(string)

	// Try to fetch execution from the database if we have an ID
	var execution *models.StageExecution
	if executionIDStr != "" {
		executionID, err := uuid.Parse(executionIDStr)
		if err == nil {
			execution, err = models.FindExecutionByID(executionID)
			if err != nil {
				log.Warnf("Couldn't find execution in database: %v, using message data", err)
			}
		}
	}

	// Prepare the payload - either from database or message
	var payload interface{}
	if execution != nil {
		// Use data from the database
		payload = map[string]interface{}{
			"id":            execution.ID.String(),
			"stage_id":      execution.StageID.String(),
			"canvas_id":     canvasID,
			"stage_event_id": execution.StageEventID.String(),
			"reference_id":  execution.ReferenceID,
			"state":         execution.State,
			"result":        execution.Result,
			"created_at":    execution.CreatedAt,
			"updated_at":    execution.UpdatedAt,
			"started_at":    execution.StartedAt,
		}
	} else {
		// Use the raw message
		payload = rawMsg
	}

	// Create the websocket event
	wsEvent := map[string]interface{}{
		"event": "execution_started",
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
		log.Debugf("Broadcasted execution_started event to canvas %s", canvasID)
	} else {
		// Fall back to broadcasting to all clients
		wsHub.BroadcastAll(wsEventJSON)
		log.Debugf("Broadcasted execution_started event to all clients")
	}
	
	return nil
}
