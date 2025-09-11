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

func HandleExecutionCancelled(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_cancelled event")

	var executionCancelled pb.StageExecutionCancelled
	if err := proto.Unmarshal(messageBody, &executionCancelled); err != nil {
		log.Errorf("Failed to unmarshal StageExecutionCancelled message: %v", err)
		return err
	}

	executionID, err := uuid.Parse(executionCancelled.ExecutionId)
	if err != nil {
		return err
	}

	execution, err := models.FindExecutionByID(executionID, uuid.MustParse(executionCancelled.StageId))
	if err != nil {
		return err
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event": "execution_cancelled",
		"payload": map[string]any{
			"id":             execution.ID.String(),
			"stage_id":       execution.StageID.String(),
			"canvas_id":      executionCancelled.CanvasId,
			"stage_event_id": execution.StageEventID.String(),
			"state":          execution.State,
			"result":         execution.Result,
			"created_at":     execution.CreatedAt,
			"updated_at":     execution.UpdatedAt,
			"started_at":     execution.FinishedAt,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(executionCancelled.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted execution_cancelled event to canvas %s", executionCancelled.CanvasId)

	return nil
}
