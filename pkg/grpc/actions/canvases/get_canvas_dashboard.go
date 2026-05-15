package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func GetCanvasDashboard(ctx context.Context, organizationID, canvasID string) (*pb.GetCanvasDashboardResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	dashboard, err := models.FindCanvasDashboard(canvasUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load canvas dashboard")
	}

	serialized, err := serializeCanvasDashboard(dashboard)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas dashboard")
	}

	return &pb.GetCanvasDashboardResponse{Dashboard: serialized}, nil
}
