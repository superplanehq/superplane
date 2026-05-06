package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateCanvasFolderMembership(
	_ context.Context,
	organizationID,
	canvasID,
	folderID string,
) (*pb.UpdateCanvasFolderMembershipResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	var folderUUID *uuid.UUID
	if folderID != "" {
		parsedFolderID, err := uuid.Parse(folderID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
		}
		folderUUID = &parsedFolderID
	}

	canvas, err := models.UpdateCanvasFolderMembership(organizationUUID, canvasUUID, folderUUID)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	serializedCanvas, err := SerializeCanvas(canvas, false, user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas")
	}

	return &pb.UpdateCanvasFolderMembershipResponse{
		Canvas: serializedCanvas,
	}, nil
}
