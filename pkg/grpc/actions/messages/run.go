package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const CanvasRunRoutingKey = "canvas-run"

type CanvasRunMessage struct {
	message *pb.CanvasRunMessage
}

func NewCanvasRunMessage(canvasID string, runID string) CanvasRunMessage {
	return CanvasRunMessage{
		message: &pb.CanvasRunMessage{
			Id:        runID,
			CanvasId:  canvasID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasRunMessage) Publish() error {
	return Publish(CanvasExchange, CanvasRunRoutingKey, toBytes(m.message))
}
