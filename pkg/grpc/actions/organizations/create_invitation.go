package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateInvitation(ctx context.Context, orgID string, email string) (*pb.CreateInvitationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	//
	// Check if user already exists in organization
	//
	_, err := models.FindUserByEmail(email, orgID)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "user is already a member of the organization")
	}

	invitation, err := models.CreateInvitation(uuid.MustParse(orgID), uuid.MustParse(userID), email)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to create invitation: %v", err)
	}

	return &pb.CreateInvitationResponse{
		Invitation: serializeInvitation(invitation),
	}, nil
}

func serializeInvitations(invitations []models.OrganizationInvitation) []*pb.Invitation {
	pbInvitations := make([]*pb.Invitation, len(invitations))

	for i, invitation := range invitations {
		pbInvitations[i] = serializeInvitation(&invitation)
	}

	return pbInvitations
}

func serializeInvitation(invitation *models.OrganizationInvitation) *pb.Invitation {
	pbInvitation := &pb.Invitation{
		Id:             invitation.ID.String(),
		OrganizationId: invitation.OrganizationID.String(),
		Email:          invitation.Email,
		Status:         string(invitation.Status),
		CreatedAt:      timestamppb.New(invitation.CreatedAt),
	}

	return pbInvitation
}
