package workflows

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteWorkflow(ctx context.Context, registry *registry.Registry, organizationID uuid.UUID, id string) (*pb.DeleteWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	workflow, err := models.FindWorkflow(organizationID, workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, templateErr := models.FindWorkflowTemplate(workflowID); templateErr == nil {
				return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
			}
		}
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	if workflow.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	// Perform soft delete on the workflow with name suffix
	// The cleanup worker will handle the actual deletion of nodes and related data
	err = workflow.SoftDelete()
	if err != nil {
		log.Errorf("failed to delete workflow %s: %v", workflow.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to delete workflow")
	}

	return &pb.DeleteWorkflowResponse{}, nil
}
