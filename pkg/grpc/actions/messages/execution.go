package messages

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ExecutionsExchange = "superplane.executions-exchange"

const (
	ExecutionPendingRoutingKey  = "execution.pending"
	ExecutionStartedRoutingKey  = "execution.started"
	ExecutionFinishedRoutingKey = "execution.finished"
)

var ExecutionRoutingKeys = []string{
	ExecutionPendingRoutingKey,
	ExecutionStartedRoutingKey,
	ExecutionFinishedRoutingKey,
}

type CanvasExecutionMessage struct {
	message *pb.CanvasNodeExecutionMessage
}

func NewCanvasExecutionMessage(canvasID, executionID, nodeID string) CanvasExecutionMessage {
	return CanvasExecutionMessage{
		message: &pb.CanvasNodeExecutionMessage{
			Id:        executionID,
			CanvasId:  canvasID,
			NodeId:    nodeID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasExecutionMessage) PublishPending() error {
	return Publish(ExecutionsExchange, ExecutionPendingRoutingKey, toBytes(m.message))
}

func (m CanvasExecutionMessage) PublishStarted() error {
	return Publish(ExecutionsExchange, ExecutionStartedRoutingKey, toBytes(m.message))
}

func (m CanvasExecutionMessage) PublishFinished() error {
	return Publish(ExecutionsExchange, ExecutionFinishedRoutingKey, toBytes(m.message))
}

func PublishCanvasExecutionState(canvasID, executionID, nodeID, state string) error {
	message := NewCanvasExecutionMessage(canvasID, executionID, nodeID)

	switch state {
	case models.CanvasNodeExecutionStatePending:
		return message.PublishPending()
	case models.CanvasNodeExecutionStateStarted:
		return message.PublishStarted()
	case models.CanvasNodeExecutionStateFinished:
		return message.PublishFinished()
	default:
		return fmt.Errorf("unknown execution state: %s", state)
	}
}

func PublishCanvasExecutionByID(workflowID, executionID uuid.UUID) error {
	execution, err := models.FindNodeExecution(workflowID, executionID)
	if err != nil {
		return fmt.Errorf("find execution: %w", err)
	}

	return PublishCanvasExecutionState(
		workflowID.String(),
		executionID.String(),
		execution.NodeID,
		execution.State,
	)
}
