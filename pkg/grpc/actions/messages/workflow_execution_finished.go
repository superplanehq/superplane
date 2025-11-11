package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowExecutionFinishedRoutingKey = "workflow-execution-finished"

type WorkflowExecutionFinishedMessage struct {
	message *pb.ExecutionFinished
}

func NewWorkflowExecutionFinishedMessage(workflowId string, executionID, nodeID string) WorkflowExecutionFinishedMessage {
	return WorkflowExecutionFinishedMessage{
		message: &pb.ExecutionFinished{
			Id:         executionID,
			WorkflowId: workflowId,
			NodeId:     nodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowExecutionFinishedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowExecutionFinishedRoutingKey, toBytes(m.message))
}
