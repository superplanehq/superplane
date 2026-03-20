package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const CanvasExecutionRoutingKey = "canvas-execution"

type CanvasExecutionMessage struct {
	message *pb.CanvasNodeExecutionMessage
}

func NewCanvasExecutionMessage(canvasId string, executionID, nodeID string) CanvasExecutionMessage {
	return CanvasExecutionMessage{
		message: &pb.CanvasNodeExecutionMessage{
			Id:        executionID,
			CanvasId:  canvasId,
			NodeId:    nodeID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasExecutionMessage) Publish() error {
	return Publish(CanvasExchange, CanvasExecutionRoutingKey, toBytes(m.message))
}
