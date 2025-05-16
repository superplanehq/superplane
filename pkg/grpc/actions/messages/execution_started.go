package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ExecutionStartedRoutingKey = "execution-started"

type ExecutionStartedMessage struct {
	message *pb.StageExecutionStarted
}

func NewExecutionStartedMessage(canvasId string, execution *models.StageExecution) ExecutionStartedMessage {
	return ExecutionStartedMessage{
		message: &pb.StageExecutionStarted{
			CanvasId:    canvasId,
			ExecutionId: execution.ID.String(),
			StageId:     execution.StageID.String(),
			EventId:     execution.StageEventID.String(),
			Timestamp:   timestamppb.Now(),
		},
	}
}

func (m ExecutionStartedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, ExecutionStartedRoutingKey, toBytes(m.message))
}
