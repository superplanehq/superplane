package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateInvitation(ctx context.Context, req *pb.CreateInvitationRequest) (*pb.CreateInvitationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Parse organization ID
	orgID, err := uuid.Parse(req.OrganizationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid organization ID")
	}

	// Validate email
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "Email is required")
	}

	// Check if user already exists in organization
	existingUser, err := models.FindUserByEmail(req.Email, orgID)
	if err == nil && existingUser.IsActive {
		return nil, status.Error(codes.AlreadyExists, "User already exists in this organization")
	}

	// Create invitation
	invitation, err := models.CreateInvitation(orgID, req.Email, uuid.MustParse(userID))
	if err != nil {
		if err == models.ErrInvitationAlreadyExists {
			return nil, status.Error(codes.AlreadyExists, "Invitation already exists for this email")
		}
		log.Errorf("Failed to create invitation: %v", err)
		return nil, status.Error(codes.Internal, "Failed to create invitation")
	}

	// Convert to response message
	invitationMsg := &pb.Invitation{
		Id:             invitation.ID.String(),
		OrganizationId: invitation.OrganizationID.String(),
		Email:          invitation.Email,
		Status:         string(invitation.Status),
		ExpiresAt:      timestamppb.New(invitation.ExpiresAt),
		CreatedAt:      timestamppb.New(invitation.CreatedAt),
	}

	log.Infof("Created invitation for %s to organization %s", req.Email, req.OrganizationId)

	return &pb.CreateInvitationResponse{
		Invitation: invitationMsg,
	}, nil
}
