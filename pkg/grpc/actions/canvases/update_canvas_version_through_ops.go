package canvases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/operations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateCanvasVersionThroughOps(
	ctx context.Context,
	registry *registry.Registry,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	versionID uuid.UUID,
	ops []*pb.CanvasUpdateOperation,
) (*pb.UpdateCanvasVersionThroughOpsResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	user, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}

	version, err := models.FindCanvasVersion(canvasID, versionID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "version not found: %v", err)
	}

	//
	// Apply operations to version.
	//
	updater := operations.NewCanvasPatcher(version, registry)
	err = updater.Patch(ops)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to update canvas version: %v", err)
	}

	//
	// Persist change to database
	//
	now := time.Now()
	newVersion := updater.GetVersion()
	newVersion.UpdatedAt = &now
	newVersion.OwnerID = &user
	err = database.Conn().Save(newVersion).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update canvas version: %v", err)
	}

	//
	// Reload version
	//
	version, err = models.FindCanvasVersion(canvasID, versionID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "version not found: %v", err)
	}

	return &pb.UpdateCanvasVersionThroughOpsResponse{
		Version: SerializeCanvasVersion(version, organizationID.String()),
	}, nil
}
