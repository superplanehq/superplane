package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ExecutionStateWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

const (
	ExecutionCreatedEvent    = "execution_created"
	ExecutionFinishedEvent   = "execution_finished"
	ExecutionStartedEvent    = "execution_started"
	ExecutionCancellingEvent = "execution_cancelling"
)

func HandleCanvasExecution(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution event")

	pbMsg := &pb.CanvasNodeExecutionMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal execution event: %w", err)
	}

	return handleExecutionState(pbMsg.CanvasId, pbMsg.Id, wsHub)
}

func workflowExecutionStateToWsEvent(workflowState string) string {
	switch workflowState {
	case models.CanvasNodeExecutionStatePending:
		return ExecutionCreatedEvent
	case models.CanvasNodeExecutionStateFinished:
		return ExecutionFinishedEvent
	case models.CanvasNodeExecutionStateStarted:
		return ExecutionStartedEvent
	case models.CanvasNodeExecutionStateCancelling:
		return ExecutionCancellingEvent
	default:
		return ""
	}
}

func handleExecutionState(workflowID string, executionID string, wsHub *ws.Hub) error {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return fmt.Errorf("failed to parse workflow id: %w", err)
	}

	executionUUID, err := uuid.Parse(executionID)
	if err != nil {
		return fmt.Errorf("failed to parse execution id: %w", err)
	}

	execution, err := models.FindNodeExecution(workflowUUID, executionUUID)
	if err != nil {
		return fmt.Errorf("failed to find execution: %w", err)
	}

	eventName := workflowExecutionStateToWsEvent(execution.State)
	if eventName == "" {
		return fmt.Errorf("unknown execution state: %s", execution.State)
	}

	resources, err := canvases.LoadNodeExecutionResources(database.Conn(), []models.CanvasNodeExecution{*execution})
	if err != nil {
		return fmt.Errorf("failed to load execution resources: %w", err)
	}

	serializedExecutions, err := canvases.SerializeNodeExecutions([]models.CanvasNodeExecution{*execution}, resources)
	if err != nil {
		return fmt.Errorf("failed to serialize execution: %w", err)
	}

	if len(serializedExecutions) == 0 {
		return fmt.Errorf("no serialized executions")
	}

	serializedExecutionJSON, err := protojson.Marshal(serializedExecutions[0])
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	event, err := json.Marshal(ExecutionStateWebsocketEvent{
		Event:   eventName,
		Payload: json.RawMessage(serializedExecutionJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(workflowID, event)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, workflowID)

	// run_started is already broadcast via the dedicated CanvasRunMessage path
	// when the root event is processed. Runs only transition to finished after
	// an execution finishes, so only check for that here.
	if execution.RunID != uuid.Nil && execution.State == models.CanvasNodeExecutionStateFinished {
		if err := handleRunState(workflowID, execution.RunID.String(), wsHub); err != nil {
			log.WithError(err).Warnf(
				"Failed to broadcast run state for execution %s in workflow %s",
				execution.ID,
				workflowID,
			)
		}
	}

	return nil
}
