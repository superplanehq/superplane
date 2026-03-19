package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	WorkflowCanvasCreatedRoutingKey        = "workflow-canvas-created"
	WorkflowCanvasUpdatedRoutingKey        = "workflow-canvas-updated"
	WorkflowCanvasVersionUpdatedRoutingKey = "workflow-canvas-version-updated"
	WorkflowCanvasDeletedRoutingKey        = "workflow-canvas-deleted"
)

type CanvasMessage struct {
	message *pb.CanvasMessage
}

type CanvasVersionMessage struct {
	message *pb.CanvasVersionMessage
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

func NewCanvasVersionUpdatedMessage(canvasID string, versionID string) CanvasVersionMessage {
	return CanvasVersionMessage{
		message: &pb.CanvasVersionMessage{
			CanvasId:  canvasID,
			VersionId: versionID,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (m CanvasMessage) PublishCreated() error {
	return Publish(WorkflowExchange, WorkflowCanvasCreatedRoutingKey, toBytes(m.message))
}

func (m CanvasMessage) PublishUpdated() error {
	return Publish(WorkflowExchange, WorkflowCanvasUpdatedRoutingKey, toBytes(m.message))
}

func (m CanvasMessage) PublishDeleted() error {
	return Publish(WorkflowExchange, WorkflowCanvasDeletedRoutingKey, toBytes(m.message))
}

func (m CanvasVersionMessage) PublishVersionUpdated() error {
	return Publish(WorkflowExchange, WorkflowCanvasVersionUpdatedRoutingKey, toBytes(m.message))
}
