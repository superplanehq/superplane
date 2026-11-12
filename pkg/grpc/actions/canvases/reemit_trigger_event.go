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
	"gorm.io/gorm"
)

func ReemitTriggerEvent(
	ctx context.Context,
	db *gorm.DB,
	canvas *models.Canvas,
	nodeID string,
	eventID uuid.UUID,
) (*pb.ReemitTriggerEventResponse, error) {
	node, err := canvas.FindNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("canvas node not found: %w", err)
	}

	if node.Ref.Data().Trigger == nil {
		return nil, fmt.Errorf("canvas node is not a trigger")
	}

	sourceEvent, err := models.FindCanvasEventForCanvas(db, canvas.ID, eventID)
	if err != nil {
		return nil, fmt.Errorf("canvas event not found: %w", err)
	}

	if sourceEvent.NodeID != nodeID {
		return nil, fmt.Errorf("event does not belong to trigger node")
	}

	if sourceEvent.ExecutionID != nil {
		return nil, fmt.Errorf("only root trigger events can be re-emitted")
	}

	now := time.Now()
	reemittedEvent := models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     nodeID,
		Channel:    sourceEvent.Channel,
		Data:       models.NewJSONValue(sourceEvent.Data.Data()),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
		CustomName: sourceEvent.CustomName,
	}

	if err := database.Conn().Create(&reemittedEvent).Error; err != nil {
		log.Errorf("failed to create re-emitted workflow event: %v", err)
		return nil, fmt.Errorf("failed to create workflow event: %w", err)
	}

	err = messages.NewCanvasEventCreatedMessage(canvas.ID.String(), canvas.OrganizationID.String(), &reemittedEvent).Publish()
	if err != nil {
		log.Errorf("failed to publish re-emitted workflow event RabbitMQ message: %v", err)
	}

	return &pb.ReemitTriggerEventResponse{
		EventId: reemittedEvent.ID.String(),
	}, nil
}
