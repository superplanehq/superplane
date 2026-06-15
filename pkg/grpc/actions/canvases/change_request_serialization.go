package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	approvals []models.CanvasChangeRequestApproval,
	usersByID map[string]*models.User,
) *pb.CanvasChangeRequest {
	var owner *pb.UserRef
	if request.OwnerID != nil {
		owner = canvasChangeRequestUserRef(request.OwnerID, usersByID)
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
		Approvals: serializeCanvasChangeRequestApprovals(approvals, usersByID),
	}

	if version != nil {
		protoRequest.Version = SerializeCanvasVersion(version, organizationID, usersByID)
	}

	return protoRequest
}

func serializeCanvasChangeRequestApprovals(
	approvals []models.CanvasChangeRequestApproval,
	usersByID map[string]*models.User,
) []*pb.CanvasChangeRequestApproval {
	serialized := make([]*pb.CanvasChangeRequestApproval, 0, len(approvals))
	for _, approval := range approvals {
		item := &pb.CanvasChangeRequestApproval{
			Approver: &pb.Canvas_ChangeManagement_Approver{
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
			item.Actor = canvasChangeRequestUserRef(approval.ActorUserID, usersByID)
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

func canvasChangeRequestUserRef(userID *uuid.UUID, usersByID map[string]*models.User) *pb.UserRef {
	if userID == nil {
		return nil
	}

	id := userID.String()
	name := ""
	if user := usersByID[id]; user != nil {
		name = user.Name
	}

	return &pb.UserRef{Id: id, Name: name}
}

func serializeCanvasChangeRequests(
	ctx context.Context,
	requests []models.CanvasChangeRequest,
	organizationID string,
) ([]*pb.CanvasChangeRequest, error) {
	var protoRequests []*pb.CanvasChangeRequest
	err := telemetry.RunSpan(ctx, "change_requests.serialize", func(ctx context.Context) error {
		data, loadErr := loadCanvasChangeRequestsSerializationData(requests, organizationID)
		if loadErr != nil {
			return loadErr
		}

		protoRequests = make([]*pb.CanvasChangeRequest, 0, len(requests))
		for i := range requests {
			request := requests[i]
			version, versionErr := versionForCanvasChangeRequestSerialization(data, request)
			if versionErr != nil {
				return versionErr
			}

			approvals := data.approvalsByRequestID[request.ID]
			protoRequests = append(protoRequests, SerializeCanvasChangeRequest(
				&request,
				version,
				organizationID,
				approvals,
				data.usersByID,
			))
		}

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(attribute.Int("change_requests.count", len(requests)))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return protoRequests, nil
}
