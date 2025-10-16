package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func RemoveSubject(ctx context.Context, authService authorization.Authorization, orgID string, canvasID string, subjectIdentifierType pbAuth.SubjectIdentifierType, subjectIdentifier string) (*pb.RemoveSubjectResponse, error) {
	if subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_ID || subjectIdentifierType == pbAuth.SubjectIdentifierType_USER_EMAIL {
		return removeUserFromCanvas(ctx, authService, orgID, canvasID, subjectIdentifierType, subjectIdentifier)
	} else if subjectIdentifierType == pbAuth.SubjectIdentifierType_INVITATION_ID {
		return removeInvitationFromCanvas(ctx, subjectIdentifier, canvasID)
	}

	return nil, status.Error(codes.InvalidArgument, "invalid subject identifier type")
}

func removeUserFromCanvas(ctx context.Context, authService authorization.Authorization, orgID string, canvasID string, subjectIdentifierType pbAuth.SubjectIdentifierType, subjectIdentifier string) (*pb.RemoveSubjectResponse, error) {
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
	roles, err := authService.GetUserRolesForCanvas(userID, canvasID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to determine user roles")
	}

	//
	// TODO: this should be in transaction
	//
	for _, role := range roles {
		err = authService.RemoveRole(user.ID.String(), role.Name, canvasID, models.DomainTypeCanvas)
		if err != nil {
			return nil, status.Error(codes.Internal, "error removing user")
		}
	}

	return &pb.RemoveSubjectResponse{}, nil
}

func removeInvitationFromCanvas(ctx context.Context, invitationID string, canvasID string) (*pb.RemoveSubjectResponse, error) {
	invitation, err := models.FindInvitationByIDWithState(invitationID, models.InvitationStatePending)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invitation not found")
	}

	// Find the canvas ID in the invitation's canvas IDs and remove it
	var updatedCanvasIDs []string
	canvasIDs := invitation.CanvasIDs.Data()

	found := false
	for _, id := range canvasIDs {
		if id != canvasID {
			updatedCanvasIDs = append(updatedCanvasIDs, id)
		} else {
			found = true
		}
	}

	if !found {
		return nil, status.Error(codes.NotFound, "invitation not associated with this canvas")
	}

	invitation.CanvasIDs = datatypes.NewJSONType(updatedCanvasIDs)
	err = models.SaveInvitation(invitation)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update invitation")
	}

	return &pb.RemoveSubjectResponse{}, nil
}
