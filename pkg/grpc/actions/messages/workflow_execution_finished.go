package messages

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WorkflowExecutionFinishedRoutingKey = "workflow-execution-finished"

type WorkflowExecutionFinishedMessage struct {
	message *pb.ExecutionFinished
}

func NewWorkflowExecutionFinishedMessage(workflowId string, execution *models.WorkflowNodeExecution) WorkflowExecutionFinishedMessage {
	return WorkflowExecutionFinishedMessage{
		message: &pb.ExecutionFinished{
			Id:         execution.ID.String(),
			WorkflowId: workflowId,
			NodeId:     execution.NodeID,
			Timestamp:  timestamppb.Now(),
		},
	}
}

func (m WorkflowExecutionFinishedMessage) Publish() error {
	return Publish(WorkflowExchange, WorkflowExecutionFinishedRoutingKey, toBytes(m.message))
}

func (m WorkflowExecutionFinishedMessage) PublishWithDelay(delay time.Duration) {
	go func() {
		time.Sleep(delay)
		err := m.Publish()
		if err != nil {
			log.Errorf("failed to publish workflow event: %v", err)
		}
	}()
}

func PublishManyWorkflowExecutionsFinishedWithDelay(workflowID string, executions []models.WorkflowNodeExecution, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		for _, execution := range executions {
			err := NewWorkflowExecutionFinishedMessage(workflowID, &execution).Publish()
			if err != nil {
				log.Errorf("failed to publish workflow event: %v", err)
			}
		}
	}()
}
