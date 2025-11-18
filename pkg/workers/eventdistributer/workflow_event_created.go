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

type WorkflowEventWebsocketEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func HandleWorkflowEventCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received execution_created event")

	pbMsg := &pb.WorkflowNodeEventMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal execution_created message: %w", err)
	}

	return handleWorkflowEventState(pbMsg.WorkflowId, pbMsg.Id, wsHub, "event_created")
}

func handleWorkflowEventState(workflowID string, eventID string, wsHub *ws.Hub, eventName string) error {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return fmt.Errorf("failed to parse workflow id: %w", err)
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("failed to parse event id: %w", err)
	}

	event, err := models.FindWorkflowEventForWorkflow(workflowUUID, eventUUID)
	if err != nil {
		return fmt.Errorf("failed to find execution: %w", err)
	}

	serializedEvent, err := workflows.SerializeWorkflowEvent(*event)
	if err != nil {
		return fmt.Errorf("failed to serialize execution: %w", err)
	}

	serializedEventJSON, err := protojson.Marshal(serializedEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	wsEvent, err := json.Marshal(WorkflowEventWebsocketEvent{
		Event:   eventName,
		Payload: json.RawMessage(serializedEventJSON),
	})

	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	wsHub.BroadcastToWorkflow(workflowID, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", eventName, workflowID)

	return nil
}
