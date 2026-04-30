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

// GetCanvasLaunchpad returns the launchpad dashboard for a canvas. When no
// launchpad row exists yet, an empty launchpad (no panels, no layout) is
// returned so the UI can render an empty grid without a separate "exists"
// check.
func GetCanvasLaunchpad(ctx context.Context, organizationID, canvasID string) (*pb.GetCanvasLaunchpadResponse, error) {
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

	launchpad, err := models.FindCanvasLaunchpad(canvasUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load canvas launchpad")
	}

	serialized, err := serializeCanvasLaunchpad(launchpad)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas launchpad")
	}

	return &pb.GetCanvasLaunchpadResponse{Launchpad: serialized}, nil
}
