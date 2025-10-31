package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowEventCreatedRoutingKey = "workflow-event-created"

type WorkflowEventCreatedMessage struct {
	message *pb.EventCreated
}

func NewWorkflowEventCreatedMessage(workflowId string, event *models.WorkflowEvent) WorkflowEventCreatedMessage {
	return WorkflowEventCreatedMessage{
		message: &pb.EventCreated{
			Id:         event.ID.String(),
			WorkflowId: workflowId,
			NodeId:     event.NodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowEventCreatedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowEventCreatedRoutingKey, toBytes(m.message))
}
