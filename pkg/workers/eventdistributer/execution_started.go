package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

// HandleExecutionStarted processes an execution started message and forwards it to websocket clients
func HandleExecutionStarted(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_started event")

	var executionStarted pb.StageExecutionStarted
	if err := proto.Unmarshal(messageBody, &executionStarted); err != nil {
		log.Warnf("Failed to unmarshal ExecutionStarted message as proto: %v, trying to continue, body: %s", err, string(messageBody))
		executionStarted = pb.StageExecutionStarted{}
	}

	// Try to fetch execution from the database if we have an ID
	var execution *models.StageExecution
	if executionStarted.ExecutionId != "" {
		executionID, err := uuid.Parse(executionStarted.ExecutionId)
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
			"id":             execution.ID.String(),
			"stage_id":       execution.StageID.String(),
			"canvas_id":      executionStarted.CanvasId,
			"stage_event_id": execution.StageEventID.String(),
			"state":          execution.State,
			"result":         execution.Result,
			"created_at":     execution.CreatedAt,
			"updated_at":     execution.UpdatedAt,
			"started_at":     execution.StartedAt,
		}
	} else {
		payload = make(map[string]interface{})
	}

	// Create the websocket event
	wsEvent := map[string]interface{}{
		"event":   "execution_started",
		"payload": payload,
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to clients
	if executionStarted.CanvasId != "" {
		// Send to specific canvas
		wsHub.BroadcastToCanvas(executionStarted.CanvasId, wsEventJSON)
		log.Debugf("Broadcasted execution_started event to canvas %s", executionStarted.CanvasId)
	} else {
		// Fall back to broadcasting to all clients
		wsHub.BroadcastAll(wsEventJSON)
		log.Debugf("Broadcasted execution_started event to all clients")
	}

	return nil
}
