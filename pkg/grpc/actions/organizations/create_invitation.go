package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func CreateInvitation(ctx context.Context, authService authorization.Authorization, orgID string, email string) (*pb.CreateInvitationResponse, error) {
	authenticatedUserID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	org := uuid.MustParse(orgID)
	authenticatedUser := uuid.MustParse(authenticatedUserID)

	//
	// Handle case where user already exists in organization,
	// either as an active user or as a deleted user (added and removed before)
	//
	user, err := models.FindMaybeDeletedUserByEmail(orgID, email)
	if err == nil {
		return handleExistingUser(authService, authenticatedUser, org, user)
	}

	//
	// Otherwise, handle case where user has never been invited to this organization.
	//
	return handleNewUser(authService, org, authenticatedUser, email)
}

func handleExistingUser(authService authorization.Authorization, authenticatedUserID, orgID uuid.UUID, user *models.User) (*pb.CreateInvitationResponse, error) {
	if !user.DeletedAt.Valid {
		return nil, status.Errorf(codes.InvalidArgument, "user %s is already an active member of organization", user.GetEmail())
	}

	var invitation *models.OrganizationInvitation
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		i, err := models.CreateInvitationInTransaction(tx, orgID, authenticatedUserID, user.GetEmail(), models.InvitationStateAccepted)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Failed to create invitation: %v", err)
		}

		invitation = i
		err = user.RestoreInTransaction(tx)
		if err != nil {
			return status.Error(codes.InvalidArgument, "Failed to restore user")
		}

		//
		// TODO: this is not using the transaction properly
		//
		return authService.AssignRole(user.ID.String(), models.RoleOrgViewer, orgID.String(), models.DomainTypeOrganization)
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateInvitationResponse{
		Invitation: serializeInvitation(invitation),
	}, nil
}

func handleNewUser(authService authorization.Authorization, orgID, userID uuid.UUID, email string) (*pb.CreateInvitationResponse, error) {
	//
	// Check if account already exists.
	// If it doesn't, we will create a pending invitation,
	// which will be fullfilled once the account signs in for the first time.
	//
	account, err := models.FindAccountByEmail(email)
	if err != nil {
		invitation, err := models.CreateInvitation(orgID, userID, email, models.InvitationStatePending)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Failed to create invitation: %v", err)
		}

		message := messages.NewInvitationCreatedMessage(invitation)
		if err := message.Publish(); err != nil {
			log.Errorf("Failed to publish invitation created message for invitation %s: %v", invitation.ID, err)
		}

		return &pb.CreateInvitationResponse{
			Invitation: serializeInvitation(invitation),
		}, nil
	}

	//
	// If an account already exists,
	// we add a new user for it to the organization immediately.
	//
	tx := database.Conn().Begin()
	invitation, err := models.CreateInvitationInTransaction(tx, orgID, userID, email, models.InvitationStateAccepted)
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.InvalidArgument, "Failed to create invitation: %v", err)
	}

	user, err := models.CreateUserInTransaction(tx, invitation.OrganizationID, account.ID, account.Email, account.Name)
	if err != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.InvalidArgument, "Failed to create user: %v", err)
	}

	//
	// TODO: this is not using the transaction properly
	//
	err = authService.AssignRole(user.ID.String(), models.RoleOrgViewer, orgID.String(), models.DomainTypeOrganization)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to commit transaction: %v", err)
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
		State:          string(invitation.State),
		CreatedAt:      timestamppb.New(invitation.CreatedAt),
	}

	return pbInvitation
}
