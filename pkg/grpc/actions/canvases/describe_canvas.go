package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
)

func loadCanvasCreator(ctx context.Context, db *gorm.DB, canvas *models.Canvas) (user *models.User, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.load_creator")
	defer done(&err)

	return models.FindMaybeDeletedUserByIDInTransaction(db.WithContext(ctx), canvas.OrganizationID.String(), canvas.CreatedBy.String())
}

func DescribeCanvas(ctx context.Context, registry *registry.Registry, organizationID string, id string) (*pb.DescribeCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	orgID := uuid.MustParse(organizationID)
	db := database.DB(ctx)

	canvas, err := loadCanvas(ctx, db, orgID, canvasID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		user, err = loadCanvasCreator(ctx, db, canvas)
		if err != nil {
			return nil, err
		}
	}

	liveVersion, err := loadLiveCanvasVersion(ctx, db, canvas)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load canvas spec")
	}

	canvasStatus, err := loadCanvasStatus(ctx, db, canvas.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load canvas status")
	}

	proto, err := serializeCanvas(ctx, canvas, liveVersion, user, canvasStatus)
	if err != nil {
		log.Errorf("failed to serialize canvas %s: %v", canvas.ID.String(), err)
		return nil, grpcerrors.Internal(err, "failed to serialize workflow")
	}

	return &pb.DescribeCanvasResponse{
		Canvas: proto,
	}, nil
}
