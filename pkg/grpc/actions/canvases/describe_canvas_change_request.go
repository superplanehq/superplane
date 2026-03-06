package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeCanvasChangeRequest(
	ctx context.Context,
	organizationID string,
	canvasID string,
	changeRequestID string,
) (*pb.DescribeCanvasChangeRequestResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	changeRequestUUID, err := uuid.Parse(changeRequestID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid change request id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	request, err := models.FindCanvasChangeRequest(canvas.ID, changeRequestUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "change request not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load change request: %v", err)
	}

	version, err := models.FindCanvasVersion(canvas.ID, request.VersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "change request version not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load change request version: %v", err)
	}

	return &pb.DescribeCanvasChangeRequestResponse{
		ChangeRequest: SerializeCanvasChangeRequest(request, version, organizationID),
	}, nil
}
