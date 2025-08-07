package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListInvitations(ctx context.Context, req *pb.ListInvitationsRequest) (*pb.ListInvitationsResponse, error) {
	orgID, err := uuid.Parse(req.OrganizationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid organization ID")
	}

	invitations, err := models.ListPendingInvitationsForOrganization(orgID)
	if err != nil {
		log.Errorf("Failed to list invitations: %v", err)
		return nil, status.Error(codes.Internal, "Failed to list invitations")
	}

	invitationMsgs := make([]*pb.Invitation, len(invitations))
	for i, invitation := range invitations {
		invitationMsgs[i] = &pb.Invitation{
			Id:             invitation.ID.String(),
			OrganizationId: invitation.OrganizationID.String(),
			Email:          invitation.Email,
			Status:         string(invitation.Status),
			ExpiresAt:      timestamppb.New(invitation.ExpiresAt),
			CreatedAt:      timestamppb.New(invitation.CreatedAt),
		}
	}

	return &pb.ListInvitationsResponse{
		Invitations: invitationMsgs,
	}, nil
}
