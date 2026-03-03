package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListCanvasChangeRequests(
	ctx context.Context,
	organizationID string,
	canvasID string,
) (*pb.ListCanvasChangeRequestsResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	requests, err := models.ListCanvasChangeRequests(canvas.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list canvas change requests: %v", err)
	}

	protoRequests := make([]*pb.CanvasChangeRequest, 0, len(requests))
	for i := range requests {
		request := requests[i]
		version, versionErr := models.FindCanvasVersion(request.WorkflowID, request.VersionID)
		if versionErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to load change request version: %v", versionErr)
		}

		protoRequests = append(protoRequests, SerializeCanvasChangeRequest(&request, version, organizationID))
	}

	return &pb.ListCanvasChangeRequestsResponse{ChangeRequests: protoRequests}, nil
}
