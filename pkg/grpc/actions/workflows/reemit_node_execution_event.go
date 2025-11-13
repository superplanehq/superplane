package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ReEmitNodeExecutionEvent(
	ctx context.Context,
	orgID uuid.UUID,
	workflowID uuid.UUID,
	nodeID string,
	executionID uuid.UUID,
) (*pb.ReEmitNodeExecutionEventResponse, error) {
	workflow, err := models.FindWorkflow(orgID, workflowID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "workflow not found")
	}

	_, err = workflow.FindNode(nodeID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "node not found")
	}

	nodeExecution, err := models.FindNodeExecutionWithNodeID(workflowID, executionID, nodeID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "node execution not found")
	}

	now := time.Now()
	newQueueItem := models.WorkflowNodeQueueItem{
		WorkflowID:  workflow.ID,
		NodeID:      nodeID,
		RootEventID: nodeExecution.RootEventID,
		EventID:     nodeExecution.EventID,
		CreatedAt:   &now,
	}

	if err := database.Conn().Create(&newQueueItem).Error; err != nil {
		log.Errorf("failed to create workflow node queue item in reemit node execution event action: %v", err)
		return nil, status.Error(codes.Internal, "failed to create workflow node queue item")
	}

	return &pb.ReEmitNodeExecutionEventResponse{
		EventId: nodeExecution.RootEventID.String(),
	}, nil
}
