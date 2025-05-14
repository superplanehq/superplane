package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/delivery"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ExecutionFinishedRoutingKey = "execution-finished"

type ExecutionFinishedMessage struct {
	message *pb.StageExecutionFinished
}

func NewExecutionFinishedMessage(canvasId string, execution *models.StageExecution) ExecutionFinishedMessage {
	return ExecutionFinishedMessage{
		message: &pb.StageExecutionFinished{
			CanvasId:    canvasId,
			ExecutionId: execution.ID.String(),
			StageId:     execution.StageID.String(),
			EventId:     execution.StageEventID.String(),
			Timestamp:   timestamppb.Now(),
		},
	}
}

func (m ExecutionFinishedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, ExecutionFinishedRoutingKey, toBytes(m.message))
}
