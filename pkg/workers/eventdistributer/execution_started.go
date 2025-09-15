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

func HandleExecutionStarted(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_started event")

	var executionStarted pb.StageExecutionStarted
	if err := proto.Unmarshal(messageBody, &executionStarted); err != nil {
		log.Errorf("Failed to unmarshal ExecutionStarted message: %v", err)
		return err
	}

	executionID, err := uuid.Parse(executionStarted.ExecutionId)
	if err != nil {
		return err
	}

	execution, err := models.FindExecutionByID(executionID, uuid.MustParse(executionStarted.StageId))
	if err != nil {
		return err
	}

	wsEventJSON, err := json.Marshal(map[string]any{
		"event": "execution_started",
		"payload": map[string]any{
			"id":             execution.ID.String(),
			"stage_id":       execution.StageID.String(),
			"canvas_id":      executionStarted.CanvasId,
			"stage_event_id": execution.StageEventID.String(),
			"state":          execution.State,
			"result":         execution.Result,
			"created_at":     execution.CreatedAt,
			"updated_at":     execution.UpdatedAt,
			"started_at":     execution.StartedAt,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToCanvas(executionStarted.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted execution_started event to canvas %s", executionStarted.CanvasId)

	return nil
}
