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

func DescribeWorkflow(ctx context.Context, registry *registry.Registry, organizationID string, id string) (*pb.DescribeWorkflowResponse, error) {
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow id: %v", err)
	}

	workflow, err := models.FindWorkflow(uuid.MustParse(organizationID), workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			template, templateErr := models.FindWorkflowTemplate(workflowID)
			if templateErr != nil {
				return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
			}
			workflow = template
		} else {
			return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
		}
	}

	proto, err := SerializeWorkflow(workflow, true)
	if err != nil {
		log.Errorf("failed to serialize workflow %s: %v", workflow.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to serialize workflow")
	}

	return &pb.DescribeWorkflowResponse{
		Workflow: proto,
	}, nil
}
