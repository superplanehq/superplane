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

func HandleExecutionCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_created event")

	pbMsg := &pb.StageExecutionStarted{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal ExecutionCreated message: %w", err)
	}

	executionID, err := uuid.Parse(pbMsg.ExecutionId)
	if err != nil {
		return fmt.Errorf("failed to parse execution ID: %w", err)
	}

	execution, err := models.FindExecutionByID(executionID)
	if err != nil {
		return fmt.Errorf("failed to find execution in database: %w", err)
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event": "execution_created",
		"payload": map[string]any{
			"id":             execution.ID.String(),
			"stage_id":       execution.StageID.String(),
			"canvas_id":      pbMsg.CanvasId,
			"stage_event_id": execution.StageEventID.String(),
			"state":          execution.State,
			"result":         execution.Result,
			"created_at":     execution.CreatedAt,
			"updated_at":     execution.UpdatedAt,
			"started_at":     execution.StartedAt,
			"finished_at":    execution.FinishedAt,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted execution_created event to canvas %s", pbMsg.CanvasId)

	return nil
}
