package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	CanvasCreatedRoutingKey        = "canvas-created"
	CanvasUpdatedRoutingKey        = "canvas-updated"
	CanvasStagingUpdatedRoutingKey = "canvas-staging-updated"
	CanvasDeletedRoutingKey        = "canvas-deleted"
	CanvasMemoryUpdatedRoutingKey  = "canvas-memory-updated"
)

type CanvasMessage struct {
	message *pb.CanvasMessage
}

type CanvasStagingMessage struct {
	message *pb.CanvasStagingMessage
}

func NewCanvasCreatedMessage(canvasID string, organizationID string) CanvasMessage {
	return CanvasMessage{
		message: &pb.CanvasMessage{
			Id:             canvasID,
			CanvasId:       canvasID,
			Timestamp:      timestamppb.Now(),
			OrganizationId: organizationID,
		},
	}
}

func NewCanvasUpdatedMessage(canvasID string, organizationID string) CanvasMessage {
	return CanvasMessage{
		message: &pb.CanvasMessage{
			Id:             canvasID,
			CanvasId:       canvasID,
			Timestamp:      timestamppb.Now(),
			OrganizationId: organizationID,
		},
	}
}

func NewCanvasDeletedMessage(canvasID string, organizationID string) CanvasMessage {
	return CanvasMessage{
		message: &pb.CanvasMessage{
			Id:             canvasID,
			CanvasId:       canvasID,
			Timestamp:      timestamppb.Now(),
			OrganizationId: organizationID,
		},
	}
}

func NewCanvasMemoryUpdatedMessage(canvasID string) CanvasMessage {
	return CanvasMessage{
		message: &pb.CanvasMessage{
			Id:        canvasID,
			CanvasId:  canvasID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func NewCanvasStagingMessage(canvasID string, userID string) CanvasStagingMessage {
	return CanvasStagingMessage{
		message: &pb.CanvasStagingMessage{
			CanvasId:  canvasID,
			UserId:    userID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasStagingMessage) Publish() error {
	return Publish(CanvasExchange, CanvasStagingUpdatedRoutingKey, toBytes(m.message))
}

func (m CanvasMessage) PublishCreated() error {
	return Publish(CanvasExchange, CanvasCreatedRoutingKey, toBytes(m.message))
}

func (m CanvasMessage) PublishUpdated() error {
	return Publish(CanvasExchange, CanvasUpdatedRoutingKey, toBytes(m.message))
}

func (m CanvasMessage) PublishDeleted() error {
	return Publish(CanvasExchange, CanvasDeletedRoutingKey, toBytes(m.message))
}

func (m CanvasMessage) PublishMemoryUpdated() error {
	return Publish(CanvasExchange, CanvasMemoryUpdatedRoutingKey, toBytes(m.message))
}
