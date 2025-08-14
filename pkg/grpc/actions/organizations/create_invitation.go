package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func CreateInvitation(ctx context.Context, authService authorization.Authorization, orgID string, email string) (*pb.CreateInvitationResponse, error) {
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
	_, err := models.FindUserByEmail(orgID, email)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "user is already a member of the organization")
	}

	org := uuid.MustParse(orgID)
	user := uuid.MustParse(userID)

	//
	// Check if account already exists.
	// If it doesn't, we will create a pending invitation,
	// which will be fullfilled once the account signs in for the first time.
	//
	account, err := models.FindAccountByEmail(email)
	if err != nil {
		invitation, err := models.CreateInvitation(org, user, email, models.InvitationStatusPending)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Failed to create invitation: %v", err)
		}

		return &pb.CreateInvitationResponse{
			Invitation: serializeInvitation(invitation),
		}, nil
	}

	//
	// If an account already exists,
	// we add a new user for it to the organization immediately.
	//
	var invitation *models.OrganizationInvitation
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		i, err := models.CreateInvitationInTransaction(tx, org, user, email, models.InvitationStatusAccepted)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Failed to create invitation: %v", err)
		}

		invitation = i
		user, err := models.CreateUserInTransaction(tx, invitation.OrganizationID, account.ID, account.Email, account.Name)
		if err != nil {
			return err
		}

		//
		// TODO: this is not using the transaction properly
		//
		return authService.AssignRole(user.ID.String(), models.RoleOrgViewer, orgID, models.DomainTypeOrganization)
	})

	if err != nil {
		return nil, err
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
