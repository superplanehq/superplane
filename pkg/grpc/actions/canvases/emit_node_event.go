package canvases

import (
	"context"
	"fmt"
	"strings"
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
		return nil, fmt.Errorf("node not found: %w", err)
	}

	now := time.Now()
	event := models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     nodeID,
		Channel:    channel,
		Data:       datatypes.NewJSONType[any](data),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	customName, err := resolveCustomName(node, data)
	if err == nil && customName != nil {
		event.CustomName = customName
	}

	if err := database.Conn().Create(&event).Error; err != nil {
		log.Errorf("failed to publish workflow event: %v", err)
		return nil, fmt.Errorf("failed to create workflow event: %w", err)
	}

	err = messages.NewCanvasEventCreatedMessage(canvasID.String(), &event).Publish()

	if err != nil {
		log.Errorf("failed to publish workflow event RabbitMQ message: %v", err)
	}

	return &pb.EmitNodeEventResponse{
		EventId: event.ID.String(),
	}, nil
}

func resolveCustomName(node *models.CanvasNode, payload map[string]any) (*string, error) {
	config := node.Configuration.Data()
	if config == nil {
		return nil, nil
	}

	rawTemplate, ok := config["customName"]
	if !ok || rawTemplate == nil {
		return nil, nil
	}

	template, ok := rawTemplate.(string)
	if !ok {
		return nil, nil
	}

	template = strings.TrimSpace(template)
	if template == "" {
		return nil, nil
	}

	builder := contexts.NewNodeConfigurationBuilder(database.Conn(), node.WorkflowID).
		WithNodeID(node.NodeID).
		WithInput(map[string]any{node.NodeID: payload})
	resolved, err := builder.ResolveExpression(template)
	if err != nil {
		return nil, err
	}

	resolvedName := strings.TrimSpace(fmt.Sprintf("%v", resolved))
	if resolvedName == "" {
		return nil, nil
	}

	return &resolvedName, nil
}
