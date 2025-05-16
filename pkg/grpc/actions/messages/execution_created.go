package messages

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ExecutionCreatedRoutingKey = "execution-created"

type ExecutionCreatedMessage struct {
	message *pb.StageExecutionCreated
}

func NewExecutionCreatedMessage(canvasId string, execution *models.StageExecution) ExecutionCreatedMessage {
	return ExecutionCreatedMessage{
		message: &pb.StageExecutionCreated{
			CanvasId:    canvasId,
			ExecutionId: execution.ID.String(),
			StageId:     execution.StageID.String(),
			EventId:     execution.StageEventID.String(),
			Timestamp:   timestamppb.Now(),
		},
	}
}

func (m ExecutionCreatedMessage) Publish() error {
	return Publish(DeliveryHubCanvasExchange, ExecutionCreatedRoutingKey, toBytes(m.message))
}
