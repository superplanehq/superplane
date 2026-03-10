package canvases

import (
	"github.com/google/uuid"
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
	case models.CanvasChangeRequestStatusRejected:
		return pb.CanvasChangeRequest_STATUS_REJECTED
	default:
		return pb.CanvasChangeRequest_STATUS_UNSPECIFIED
	}
}

func canvasChangeRequestApprovalStateToProto(state string) pb.CanvasChangeRequestApproval_State {
	switch state {
	case models.CanvasChangeRequestApprovalStateApproved:
		return pb.CanvasChangeRequestApproval_STATE_APPROVED
	case models.CanvasChangeRequestApprovalStateRejected:
		return pb.CanvasChangeRequestApproval_STATE_REJECTED
	case models.CanvasChangeRequestApprovalStateUnapproved:
		return pb.CanvasChangeRequestApproval_STATE_UNAPPROVED
	default:
		return pb.CanvasChangeRequestApproval_STATE_UNSPECIFIED
	}
}

func SerializeCanvasChangeRequest(
	request *models.CanvasChangeRequest,
	version *models.CanvasVersion,
	organizationID string,
) *pb.CanvasChangeRequest {
	approvals, err := models.ListCanvasChangeRequestApprovals(request.WorkflowID, request.ID)
	if err != nil {
		approvals = nil
	}

	var owner *pb.UserRef
	if request.OwnerID != nil {
		owner = findCanvasChangeRequestUserRef(organizationID, request.OwnerID)
	}

	metadata := &pb.CanvasChangeRequest_Metadata{
		Id:           request.ID.String(),
		CanvasId:     request.WorkflowID.String(),
		VersionId:    request.VersionID.String(),
		Owner:        owner,
		Status:       canvasChangeRequestStatusToProto(request.Status),
		Title:        request.Title,
		Description:  request.Description,
		IsConflicted: request.IsConflicted(),
	}
	if request.BasedOnVersionID != nil {
		metadata.BasedOnVersionId = request.BasedOnVersionID.String()
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
			ChangedNodeIds:     request.ChangedNodeIDs,
			ConflictingNodeIds: request.ConflictingNodeIDs,
		},
		Approvals: serializeCanvasChangeRequestApprovals(organizationID, approvals),
	}

	if version != nil {
		protoRequest.Version = SerializeCanvasVersion(version, organizationID)
	}

	return protoRequest
}

func serializeCanvasChangeRequestApprovals(
	organizationID string,
	approvals []models.CanvasChangeRequestApproval,
) []*pb.CanvasChangeRequestApproval {
	serialized := make([]*pb.CanvasChangeRequestApproval, 0, len(approvals))
	for _, approval := range approvals {
		item := &pb.CanvasChangeRequestApproval{
			Approver: &pb.CanvasChangeRequestApprover{
				Type: canvasChangeRequestApproverTypeToProto(approval.ApproverType),
			},
			State: canvasChangeRequestApprovalStateToProto(approval.State),
		}

		if approval.ApproverUserID != nil {
			item.Approver.UserId = approval.ApproverUserID.String()
		}
		if approval.ApproverRole != nil {
			item.Approver.RoleName = *approval.ApproverRole
		}
		if approval.ActorUserID != nil {
			item.Actor = findCanvasChangeRequestUserRef(organizationID, approval.ActorUserID)
		}
		if approval.CreatedAt != nil {
			item.CreatedAt = timestamppb.New(*approval.CreatedAt)
		}
		if approval.InvalidatedAt != nil {
			item.InvalidatedAt = timestamppb.New(*approval.InvalidatedAt)
		}

		serialized = append(serialized, item)
	}

	return serialized
}

func findCanvasChangeRequestUserRef(organizationID string, userID *uuid.UUID) *pb.UserRef {
	if userID == nil {
		return nil
	}

	id := userID.String()
	name := ""
	if user, err := models.FindMaybeDeletedUserByID(organizationID, id); err == nil && user != nil {
		name = user.Name
	}

	return &pb.UserRef{Id: id, Name: name}
}
