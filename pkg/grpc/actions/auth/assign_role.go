package auth

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func AssignRole(ctx context.Context, orgID, domainType, domainID, roleName string, subjectIdentifierType pbAuth.SubjectIdentifierType, subjectIdentifier string, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_ID || subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_EMAIL {
		return assignRoleToUser(authService, subjectIdentifierType, subjectIdentifier, orgID, roleName, domainID, domainType)
	} else if subjectIdentifierType == pbAuth.SubjectIdentifierType_INVITATION_ID {
		return assignRoleToInvitation(subjectIdentifier, roleName, domainID, domainType)
	}

	return nil, status.Error(codes.InvalidArgument, "invalid subject identifier type")
}

func assignRoleToUser(authService authorization.Authorization, subjectIdentifierType pbAuth.SubjectIdentifierType, subjectIdentifier, orgID, roleName, domainID, domainType string) (*pb.AssignRoleResponse, error) {
	var user *models.User
	var err error

	if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_ID {
		user, err = models.FindActiveUserByID(orgID, subjectIdentifier)
	} else if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_EMAIL {
		user, err = models.FindActiveUserByEmail(orgID, subjectIdentifier)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user not found")
	}
	err = authService.AssignRole(user.ID.String(), roleName, domainID, domainType)
	if err != nil {
		log.Errorf("Error assigning role %s to user %s: %v", roleName, user.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}

func assignRoleToInvitation(invitationID, roleName, domainID, domainType string) (*pb.AssignRoleResponse, error) {
	invitation, err := models.FindInvitationByIDWithState(invitationID, models.InvitationStatePending)
	if err != nil {
		log.Errorf("Invitation not found: %s", invitationID)
		return nil, status.Error(codes.NotFound, "invitation not found")
	}

	if domainType != models.DomainTypeCanvas {
		return nil, status.Error(codes.InvalidArgument, "only canvas roles can be assigned to invitations")
	}

	if roleName != models.RoleCanvasViewer {
		return nil, status.Error(codes.InvalidArgument, "only canvas viewer role can be assigned to invitations")
	}

	canvasID, err := uuid.Parse(domainID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	canvasIDStr := canvasID.String()
	canvasIDs := invitation.CanvasIDs.Data()
	for _, existingCanvasID := range canvasIDs {
		if existingCanvasID == canvasIDStr {
			return &pb.AssignRoleResponse{}, nil
		}
	}

	canvasIDs = append(canvasIDs, canvasIDStr)
	invitation.CanvasIDs = datatypes.NewJSONType(canvasIDs)

	err = models.SaveInvitation(invitation)
	if err != nil {
		log.Errorf("Error updating invitation %s with canvas ID %s: %v", invitationID, canvasID, err)
		return nil, status.Error(codes.Internal, "failed to assign canvas role to invitation")
	}

	return &pb.AssignRoleResponse{}, nil
}
