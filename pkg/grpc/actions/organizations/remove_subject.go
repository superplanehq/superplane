package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveSubject(ctx context.Context, authService authorization.Authorization, orgID string, subjectIdentifierType pbAuth.SubjectIdentifierType, subjectIdentifier string) (*pb.RemoveSubjectResponse, error) {
	if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_ID || subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_EMAIL {
		return removeUser(ctx, authService, orgID, subjectIdentifierType, subjectIdentifier)
	} else if subjectIdentifierType == pbAuth.SubjectIdentifierType_INVITATION_ID {
		return removeInvitation(ctx, subjectIdentifier)
	}

	return nil, status.Error(codes.InvalidArgument, "invalid subject identifier type")
}

func removeUser(ctx context.Context, authService authorization.Authorization, orgID string, subjectIdentifierType pbAuth.SubjectIdentifierType, subjectIdentifier string) (*pb.RemoveSubjectResponse, error) {
	var user *models.User
	var err error

	if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_ID {
		user, err = models.FindActiveUserByID(orgID, subjectIdentifier)
	} else if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_EMAIL {
		user, err = models.FindActiveUserByEmail(orgID, subjectIdentifier)
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	userID := user.ID.String()

	//
	// TODO: this should all be inside of a transaction
	// Remove the access to all the canvases first
	//
	canvases, err := authService.GetAccessibleCanvasesForUser(userID)
	if err != nil {
		log.Errorf("Error getting accessible canvases for %s: %v", userID, err)
		return nil, status.Error(codes.Internal, "error getting accessible canvases")
	}

	for _, canvas := range canvases {
		roles, err := authService.GetUserRolesForCanvas(userID, canvas)
		if err != nil {
			log.Errorf("Error getting user roles for canvas %s: %v", canvas, err)
			return nil, status.Error(codes.Internal, "error removing access to canvases")
		}

		for _, role := range roles {
			err = authService.RemoveRole(userID, role.Name, canvas, models.DomainTypeCanvas)
			if err != nil {
				log.Errorf("Error removing role %s for %s: %v", role.Name, userID, err)
				return nil, status.Error(codes.Internal, "error removing role")
			}
		}
	}

	//
	// Remove organization roles
	//
	roles, err := authService.GetUserRolesForOrg(user.ID.String(), orgID)
	if err != nil {
		log.Errorf("Error determing user roles for %s: %v", user.ID.String(), err)
		return nil, status.Error(codes.Internal, "error determing user roles")
	}

	for _, role := range roles {
		err = authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization)
		if err != nil {
			log.Errorf("Error removing role %s for %s: %v", role.Name, user.ID.String(), err)
			return nil, status.Error(codes.Internal, "error removing role")
		}
	}

	err = user.Delete()
	if err != nil {
		return nil, status.Error(codes.Internal, "error deleting user")
	}

	return &pb.RemoveSubjectResponse{}, nil
}

func removeInvitation(ctx context.Context, invitationID string) (*pb.RemoveSubjectResponse, error) {
	invitation, err := models.FindInvitationByIDWithState(invitationID, models.InvitationStatePending)
	if err != nil {
		log.Errorf("Invitation not found: %s", invitationID)
		return nil, status.Error(codes.NotFound, "invitation not found")
	}

	err = invitation.Delete()
	if err != nil {
		log.Errorf("Error deleting invitation %s: %v", invitationID, err)
		return nil, status.Error(codes.Internal, "failed to delete invitation")
	}

	return &pb.RemoveSubjectResponse{}, nil
}
