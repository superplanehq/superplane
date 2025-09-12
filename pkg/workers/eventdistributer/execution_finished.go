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

func HandleExecutionFinished(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_finished event")

	var executionFinished pb.StageExecutionFinished
	if err := proto.Unmarshal(messageBody, &executionFinished); err != nil {
		log.Errorf("Failed to unmarshal ExecutionFinished message: %v", err)
		return err
	}

	executionID, err := uuid.Parse(executionFinished.ExecutionId)
	if err != nil {
		return err
	}

	execution, err := models.FindExecutionByID(executionID, uuid.MustParse(executionFinished.StageId))
	if err != nil {
		return err
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event": "execution_finished",
		"payload": map[string]any{
			"id":             execution.ID.String(),
			"stage_id":       execution.StageID.String(),
			"canvas_id":      executionFinished.CanvasId,
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

	wsHub.BroadcastToCanvas(executionFinished.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted execution_finished event to canvas %s", executionFinished.CanvasId)

	return nil
}
