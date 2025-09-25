package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ExecutionCancelledRoutingKey = "execution-cancelled"

type ExecutionCancelledMessage struct {
	message *pb.StageExecutionCancelled
}

func NewExecutionCancelledMessage(canvasId string, execution *models.StageExecution) ExecutionCancelledMessage {
	return ExecutionCancelledMessage{
		message: &pb.StageExecutionCancelled{
			CanvasId:    canvasId,
			ExecutionId: execution.ID.String(),
			StageId:     execution.StageID.String(),
			EventId:     execution.StageEventID.String(),
			Timestamp:   timestamppb.Now(),
		},
	}
}

func (m ExecutionCancelledMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, ExecutionCancelledRoutingKey, toBytes(m.message))
}
