package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

// HandleExecutionCreated processes an execution created message and forwards it to websocket clients
func HandleExecutionCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_created event")

	// Parse the protobuf message
	pbMsg := &pb.StageExecutionStarted{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal ExecutionCreated message: %w", err)
	}

	// Fetch execution information directly from the database
	executionID, err := uuid.Parse(pbMsg.ExecutionId)
	if err != nil {
		return fmt.Errorf("failed to parse execution ID: %w", err)
	}

	execution, err := models.FindExecutionByID(executionID)
	if err != nil {
		return fmt.Errorf("failed to find execution in database: %w", err)
	}

	// Found execution in database, convert to a WebSocket-friendly format
	wsEvent := map[string]interface{}{
		"event": "execution_created",
		"payload": map[string]interface{}{
			"id":             execution.ID.String(),
			"stage_id":       execution.StageID.String(),
			"canvas_id":      pbMsg.CanvasId, // Use from message as it might not be in the model
			"stage_event_id": execution.StageEventID.String(),
			"state":          execution.State,
			"result":         execution.Result,
			"created_at":     execution.CreatedAt,
			"updated_at":     execution.UpdatedAt,
			"started_at":     execution.StartedAt,
			"finished_at":    execution.FinishedAt,
		},
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to all clients subscribed to this canvas
	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted execution_created event to canvas %s", pbMsg.CanvasId)

	return nil
}
