package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowExecutionRoutingKey = "workflow-execution"

type WorkflowExecutionMessage struct {
	message *pb.WorkflowNodeExecutionMessage
}

func NewWorkflowExecutionMessage(workflowId string, executionID, nodeID string) WorkflowExecutionMessage {
	return WorkflowExecutionMessage{
		message: &pb.WorkflowNodeExecutionMessage{
			Id:         executionID,
			WorkflowId: workflowId,
			NodeId:     nodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowExecutionMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowExecutionRoutingKey, toBytes(m.message))
}
