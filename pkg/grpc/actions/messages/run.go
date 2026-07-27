package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	CanvasRunRoutingKey  = "canvas-run"
	RunPendingRoutingKey = "run.pending"
)

type CanvasRunMessage struct {
	message *pb.CanvasRunMessage
}

func NewCanvasRunMessage(canvasID, runID string) CanvasRunMessage {
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

func (m CanvasRunMessage) PublishPending() error {
	return Publish(CanvasExchange, RunPendingRoutingKey, toBytes(m.message))
}
