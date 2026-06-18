package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeCanvas(ctx context.Context, registry *registry.Registry, organizationID string, id string) (*pb.DescribeCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgID := uuid.MustParse(organizationID)
	canvas, err := loadCanvas(ctx, orgID, canvasID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		err = telemetry.RunSpan(ctx, "canvases.load_creator", func(ctx context.Context) error {
			var loadErr error
			user, loadErr = models.FindMaybeDeletedUserByIDInTransaction(database.DB(ctx), canvas.OrganizationID.String(), canvas.CreatedBy.String())
			return loadErr
		})
		if err != nil {
			return nil, err
		}
	}

	proto, err := serializeCanvas(ctx, canvas, true, user)
	if err != nil {
		log.Errorf("failed to serialize canvas %s: %v", canvas.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to serialize workflow")
	}

	return &pb.DescribeCanvasResponse{
		Canvas: proto,
	}, nil
}
