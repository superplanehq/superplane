package canvases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
)

func EmitNodeEvent(
	ctx context.Context,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	nodeID string,
	channel string,
	data map[string]any,
) (*pb.EmitNodeEventResponse, error) {
	canvas, err := models.FindCanvas(orgID, canvasID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	node, err := canvas.FindNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("canvas node not found: %w", err)
	}

	now := time.Now()
	event := models.CanvasEvent{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     nodeID,
		Channel:    channel,
		Data:       datatypes.NewJSONType[any](data),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	rootPayload := contexts.BuildRootEventPayload(data, "", now)
	runTitle, err := contexts.ResolveRootEventRunTitle(
		database.Conn(),
		node,
		rootPayload,
		rootPayload,
	)
	if err == nil && runTitle != nil {
		event.RunTitle = runTitle
	}

	if err := database.Conn().Create(&event).Error; err != nil {
		log.Errorf("failed to publish workflow event: %v", err)
		return nil, fmt.Errorf("failed to create workflow event: %w", err)
	}

	err = messages.NewCanvasEventCreatedMessage(canvasID.String(), canvas.OrganizationID.String(), &event).Publish()

	if err != nil {
		log.Errorf("failed to publish workflow event RabbitMQ message: %v", err)
	}

	return &pb.EmitNodeEventResponse{
		EventId: event.ID.String(),
	}, nil
}
