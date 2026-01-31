package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeCanvas(ctx context.Context, registry *registry.Registry, organizationID string, id string) (*pb.DescribeCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindWorkflow(uuid.MustParse(organizationID), canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			template, templateErr := models.FindWorkflowTemplate(canvasID)
			if templateErr != nil {
				return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
			}
			canvas = template
		} else {
			return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
		}
	}

	proto, err := SerializeCanvas(canvas, true)
	if err != nil {
		log.Errorf("failed to serialize canvas %s: %v", canvas.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to serialize workflow")
	}

	return &pb.DescribeCanvasResponse{
		Canvas: proto,
	}, nil
}
