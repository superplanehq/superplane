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

func UpdateCanvasFolder(
	_ context.Context,
	organizationID string,
	req *pb.UpdateCanvasFolderRequest,
) (*pb.UpdateCanvasFolderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder update request is required")
	}

	switch operation := req.Operation.(type) {
	case *pb.UpdateCanvasFolderRequest_Update:
		return updateCanvasFolderMetadata(organizationID, req.Id, operation.Update)
	case *pb.UpdateCanvasFolderRequest_Move:
		return updateCanvasFolderPosition(organizationID, req.Id, operation.Move)
	case *pb.UpdateCanvasFolderRequest_Membership:
		return updateCanvasFolderMembership(organizationID, req.Id, operation.Membership)
	default:
		return nil, status.Error(codes.InvalidArgument, "canvas folder update operation is required")
	}
}

func updateCanvasFolderMetadata(
	organizationID,
	id string,
	update *pb.UpdateCanvasFolderFields,
) (*pb.UpdateCanvasFolderResponse, error) {
	organizationUUID, folderID, err := parseCanvasFolderUpdateIDs(organizationID, id)
	if err != nil {
		return nil, err
	}

	if update == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder update operation is required")
	}

	return updateCanvasFolderFields(organizationUUID, folderID, update.Folder)
}

func updateCanvasFolderPosition(
	organizationID,
	id string,
	move *pb.MoveCanvasFolder,
) (*pb.UpdateCanvasFolderResponse, error) {
	organizationUUID, folderID, err := parseCanvasFolderUpdateIDs(organizationID, id)
	if err != nil {
		return nil, err
	}

	if move == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder move operation is required")
	}

	if move.Direction == pb.UpdateCanvasFolderRequest_DIRECTION_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "canvas folder move direction is required")
	}

	return moveCanvasFolder(organizationUUID, folderID, move.Direction)
}

func parseCanvasFolderUpdateIDs(organizationID, folderID string) (uuid.UUID, uuid.UUID, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	return organizationUUID, folderUUID, nil
}

func moveCanvasFolder(
	organizationID,
	folderID uuid.UUID,
	direction pb.UpdateCanvasFolderRequest_Direction,
) (*pb.UpdateCanvasFolderResponse, error) {
	folders, err := models.MoveCanvasFolder(organizationID, folderID, direction.String())
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.UpdateCanvasFolderResponse{
		Folders: SerializeCanvasFolders(folders),
	}, nil
}

func updateCanvasFolderFields(
	organizationID,
	folderID uuid.UUID,
	folder *pb.CanvasFolder,
) (*pb.UpdateCanvasFolderResponse, error) {
	if folder == nil || folder.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas folder is required")
	}

	updatedFolder, err := models.UpdateCanvasFolder(organizationID, folderID, folder.Spec.Title, folder.Spec.BackgroundColor)
	if err != nil {
		return nil, canvasFolderErrorToStatus(err)
	}

	return &pb.UpdateCanvasFolderResponse{
		Folder: SerializeCanvasFolder(updatedFolder),
	}, nil
}

func updateCanvasFolderMembership(
	organizationID,
	id string,
	membership *pb.UpdateCanvasFolderMembership,
) (*pb.UpdateCanvasFolderResponse, error) {
	if id != "" {
		return nil, status.Error(codes.InvalidArgument, "canvas folder id cannot be set while updating canvas membership")
	}

	organizationUUID, canvasIDs, folderUUID, err := parseCanvasFolderMembershipIDs(organizationID, membership)
	if err != nil {
		return nil, err
	}

	canvases := make([]*models.Canvas, 0, len(canvasIDs))
	for _, canvasID := range canvasIDs {
		canvas, err := models.UpdateCanvasFolderMembership(organizationUUID, canvasID, folderUUID)
		if err != nil {
			return nil, canvasFolderErrorToStatus(err)
		}

		publishCanvasUpdated(canvas)
		canvases = append(canvases, canvas)
	}

	return serializeCanvasFolderMembershipResponse(canvases)
}

func parseCanvasFolderMembershipIDs(
	organizationID string,
	membership *pb.UpdateCanvasFolderMembership,
) (uuid.UUID, []uuid.UUID, *uuid.UUID, error) {
	if membership == nil {
		return uuid.Nil, nil, nil, status.Error(codes.InvalidArgument, "canvas folder membership operation is required")
	}

	if len(membership.CanvasIds) == 0 {
		return uuid.Nil, nil, nil, status.Error(codes.InvalidArgument, "at least one canvas id is required")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return uuid.Nil, nil, nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvasIDs, err := parseCanvasIDs(membership.CanvasIds)
	if err != nil {
		return uuid.Nil, nil, nil, err
	}

	folderUUID, err := parseOptionalCanvasFolderID(membership.FolderId)
	if err != nil {
		return uuid.Nil, nil, nil, err
	}

	return organizationUUID, canvasIDs, folderUUID, nil
}

func parseCanvasIDs(canvasIDs []string) ([]uuid.UUID, error) {
	parsedCanvasIDs := make([]uuid.UUID, 0, len(canvasIDs))
	seenCanvasIDs := map[uuid.UUID]struct{}{}

	for _, canvasID := range canvasIDs {
		canvasUUID, err := uuid.Parse(canvasID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
		}

		if _, ok := seenCanvasIDs[canvasUUID]; ok {
			return nil, status.Error(codes.InvalidArgument, "canvas ids cannot contain duplicates")
		}

		seenCanvasIDs[canvasUUID] = struct{}{}
		parsedCanvasIDs = append(parsedCanvasIDs, canvasUUID)
	}

	return parsedCanvasIDs, nil
}

func parseOptionalCanvasFolderID(folderID string) (*uuid.UUID, error) {
	if folderID == "" {
		return nil, nil
	}

	parsedFolderID, err := uuid.Parse(folderID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas folder id: %v", err)
	}

	return &parsedFolderID, nil
}

func publishCanvasUpdated(canvas *models.Canvas) {
	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}
}

func serializeCanvasFolderMembershipResponse(canvases []*models.Canvas) (*pb.UpdateCanvasFolderResponse, error) {
	serializedCanvases := make([]*pb.Canvas, 0, len(canvases))
	for _, canvas := range canvases {
		user, err := findCanvasCreator(canvas)
		if err != nil {
			return nil, err
		}

		serializedCanvas, err := SerializeCanvas(canvas, false, user)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to serialize canvas")
		}

		serializedCanvases = append(serializedCanvases, serializedCanvas)
	}

	response := &pb.UpdateCanvasFolderResponse{Canvases: serializedCanvases}
	if len(serializedCanvases) == 1 {
		response.Canvas = serializedCanvases[0]
	}

	return response, nil
}

func findCanvasCreator(canvas *models.Canvas) (*models.User, error) {
	if canvas.CreatedBy == nil {
		return nil, nil
	}

	return models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
}
