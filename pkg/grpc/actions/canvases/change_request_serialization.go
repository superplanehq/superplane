package canvases

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func canvasChangeRequestStatusToProto(status string) pb.CanvasChangeRequest_Status {
	switch status {
	case models.CanvasChangeRequestStatusOpen:
		return pb.CanvasChangeRequest_STATUS_OPEN
	case models.CanvasChangeRequestStatusPublished:
		return pb.CanvasChangeRequest_STATUS_PUBLISHED
	default:
		return pb.CanvasChangeRequest_STATUS_UNSPECIFIED
	}
}

func SerializeCanvasChangeRequest(
	request *models.CanvasChangeRequest,
	version *models.CanvasVersion,
	organizationID string,
) *pb.CanvasChangeRequest {
	var owner *pb.UserRef
	if request.OwnerID != nil {
		ownerID := request.OwnerID.String()
		ownerName := ""
		if user, err := models.FindMaybeDeletedUserByID(organizationID, ownerID); err == nil && user != nil {
			ownerName = user.Name
		}
		owner = &pb.UserRef{Id: ownerID, Name: ownerName}
	}

	metadata := &pb.CanvasChangeRequest_Metadata{
		Id:          request.ID.String(),
		CanvasId:    request.WorkflowID.String(),
		VersionId:   request.VersionID.String(),
		Owner:       owner,
		Status:      canvasChangeRequestStatusToProto(request.Status),
		Title:       request.Title,
		Description: request.Description,
	}

	if request.PublishedAt != nil {
		metadata.PublishedAt = timestamppb.New(*request.PublishedAt)
	}
	if request.CreatedAt != nil {
		metadata.CreatedAt = timestamppb.New(*request.CreatedAt)
	}
	if request.UpdatedAt != nil {
		metadata.UpdatedAt = timestamppb.New(*request.UpdatedAt)
	}

	protoRequest := &pb.CanvasChangeRequest{
		Metadata: metadata,
		Diff: &pb.CanvasChangeRequestDiff{
			ChangedNodeIds: request.ChangedNodeIDs,
		},
	}

	if version != nil {
		protoRequest.Version = SerializeCanvasVersion(version, organizationID)
	}

	return protoRequest
}
