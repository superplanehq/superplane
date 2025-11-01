package messages

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowExecutionCreatedRoutingKey = "workflow-execution-created"

type WorkflowExecutionCreatedMessage struct {
	message *pb.ExecutionCreated
}

func NewWorkflowExecutionCreatedMessage(workflowId string, execution *models.WorkflowNodeExecution) WorkflowExecutionCreatedMessage {
	return WorkflowExecutionCreatedMessage{
		message: &pb.ExecutionCreated{
			Id:         execution.ID.String(),
			WorkflowId: workflowId,
			NodeId:     execution.NodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowExecutionCreatedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowExecutionCreatedRoutingKey, toBytes(m.message))
}

func (m WorkflowExecutionCreatedMessage) PublishWithDelay(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		err := m.Publish()
		if err != nil {
			log.Errorf("failed to publish workflow event: %v", err)
		}
	}()
}

func PublishManyWorkflowExecutionsWithDelay(workflowID string, executions []models.WorkflowNodeExecution, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		for _, execution := range executions {
			err := NewWorkflowExecutionCreatedMessage(workflowID, &execution).Publish()
			if err != nil {
				log.Errorf("failed to publish workflow event: %v", err)
			}
		}
	}()
}
