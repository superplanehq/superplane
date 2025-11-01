package messages

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowExecutionStartedRoutingKey = "workflow-execution-started"

type WorkflowExecutionStartedMessage struct {
	message *pb.ExecutionStarted
}

func NewWorkflowExecutionStartedMessage(workflowId string, execution *models.WorkflowNodeExecution) WorkflowExecutionStartedMessage {
	return WorkflowExecutionStartedMessage{
		message: &pb.ExecutionStarted{
			Id:         execution.ID.String(),
			WorkflowId: workflowId,
			NodeId:     execution.NodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowExecutionStartedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowExecutionStartedRoutingKey, toBytes(m.message))
}

func (m WorkflowExecutionStartedMessage) PublishWithDelay(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		err := m.Publish()
		if err != nil {
			log.Errorf("failed to publish workflow event: %v", err)
		}
	}()
}

func PublishManyWorkflowExecutionsStartedWithDelay(workflowID string, executions []models.WorkflowNodeExecution, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		for _, execution := range executions {
			err := NewWorkflowExecutionStartedMessage(workflowID, &execution).Publish()
			if err != nil {
				log.Errorf("failed to publish workflow event: %v", err)
			}
		}
	}()
}
