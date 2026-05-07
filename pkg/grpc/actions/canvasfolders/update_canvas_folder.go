package canvasfolders

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateCanvasFolder(
	_ context.Context,
	organizationID,
	id string,
	folder *pb.CanvasFolder,
	replaceMembership bool,
) (*pb.UpdateCanvasFolderResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folderID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	if folder == nil || folder.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder is required")
	}

	updatedFolder, err := updateCanvasFolder(organizationUUID, folderID, folder, replaceMembership)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err, "failed to update canvas folder")
	}

	return &pb.UpdateCanvasFolderResponse{
		Folder: SerializeCanvasFolder(updatedFolder),
	}, nil
}

func updateCanvasFolder(organizationID, folderID uuid.UUID, folder *pb.CanvasFolder, replaceMembership bool) (*models.CanvasFolder, error) {
	if !replaceMembership {
		return models.UpdateCanvasFolder(organizationID, folderID, folder.Spec.Title, folder.Spec.BackgroundColor)
	}

	canvasIDs, err := parseCanvasFolderMembership(folder.Spec.Canvases)
	if err != nil {
		return nil, err
	}

	updatedFolder, affectedCanvases, err := models.UpdateCanvasFolderWithMembership(
		organizationID,
		folderID,
		folder.Spec.Title,
		folder.Spec.BackgroundColor,
		canvasIDs,
	)
	if err != nil {
		return nil, err
	}

	for _, canvas := range affectedCanvases {
		if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); publishErr != nil {
			log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
		}
	}

	return updatedFolder, nil
}

func parseCanvasFolderMembership(canvases []*pb.CanvasRef) ([]uuid.UUID, error) {
	canvasIDs := make([]uuid.UUID, 0, len(canvases))
	for _, canvas := range canvases {
		if canvas == nil {
			return nil, status.Error(codes.InvalidArgument, "canvas id is required")
		}

		id, err := uuid.Parse(canvas.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
		}

		canvasIDs = append(canvasIDs, id)
	}

	return canvasIDs, nil
}
