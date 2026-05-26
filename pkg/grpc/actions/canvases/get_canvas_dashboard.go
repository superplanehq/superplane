package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func GetCanvasDashboard(ctx context.Context, organizationID, canvasID string, versionID string) (*pb.GetCanvasDashboardResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	var dashboard *models.CanvasDashboard
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		resolvedVersionID, resolveErr := resolveDashboardVersionID(tx, canvas, strings.TrimSpace(versionID))
		if resolveErr != nil {
			return resolveErr
		}

		version, loadErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, resolvedVersionID)
		if loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return loadErr
		}

		if accessErr := ensureDashboardVersionReadable(ctx, tx, canvas, version); accessErr != nil {
			return accessErr
		}

		dashboard = models.CanvasDashboardFromVersion(version)
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Error(codes.Internal, "failed to load canvas dashboard")
	}

	serialized, err := serializeCanvasDashboard(dashboard)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas dashboard")
	}

	return &pb.GetCanvasDashboardResponse{Dashboard: serialized}, nil
}
