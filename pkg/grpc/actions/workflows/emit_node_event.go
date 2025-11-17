package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"gorm.io/datatypes"
)

func EmitNodeEvent(
	ctx context.Context,
	orgID uuid.UUID,
	workflowID uuid.UUID,
	nodeID string,
	channel string,
	data map[string]any,
) (*pb.EmitNodeEventResponse, error) {
	workflow, err := models.FindWorkflow(orgID, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	_, err = workflow.FindNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	now := time.Now()
	event := models.WorkflowEvent{
		WorkflowID: workflow.ID,
		NodeID:     nodeID,
		Channel:    channel,
		Data:       datatypes.NewJSONType[any](data),
		State:      models.WorkflowEventStatePending,
		CreatedAt:  &now,
	}

	if err := database.Conn().Create(&event).Error; err != nil {
		return nil, fmt.Errorf("failed to create workflow event: %w", err)
	}

	if err != nil {
		log.Errorf("failed to publish workflow event: %v", err)
	}

	return &pb.EmitNodeEventResponse{
		EventId: event.ID.String(),
	}, nil
}
