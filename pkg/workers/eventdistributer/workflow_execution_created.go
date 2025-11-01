package eventdistributer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/workflows"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ExecutionStateWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleWorkflowExecutionCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_created event")

	pbMsg := &pb.ExecutionCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal execution_created message: %w", err)
	}

	return handleExecutionState(pbMsg.WorkflowId, pbMsg.Id, wsHub, "execution_created")
}

func handleExecutionState(workflowID string, executionID string, wsHub *ws.Hub, eventName string) error {
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

	serializedExecutions, err := workflows.SerializeNodeExecutions([]models.WorkflowNodeExecution{*execution}, []models.WorkflowNodeExecution{})
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

	return nil
}
