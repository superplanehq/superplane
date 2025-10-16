package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func UpdateInvitation(ctx context.Context, authService authorization.Authorization, orgID string, invitationID string, canvasIDs []string) (*organizations.UpdateInvitationResponse, error) {
	invitation, err := models.FindInvitationByIDWithState(invitationID, models.InvitationStatePending)
	if err != nil {
		log.Errorf("Invitation not found: %s", invitationID)
		return nil, status.Error(codes.NotFound, "invitation not found")
	}

	if len(canvasIDs) > 0 {
		if err := actions.ValidateUUIDsArray(canvasIDs); err != nil {
			return nil, err
		}

		parsedCanvasIDs := []uuid.UUID{}
		for _, canvasID := range canvasIDs {
			parsedCanvasIDs = append(parsedCanvasIDs, uuid.MustParse(canvasID))
		}

		exists, err := models.ExistManyCanvases(uuid.MustParse(orgID), parsedCanvasIDs)
		if err != nil {
			log.Errorf("Error checking canvas existence: %v", err)
			return nil, status.Error(codes.Internal, "failed to check canvas existence")
		}

		if !exists {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
	}

	invitation.CanvasIDs = datatypes.NewJSONType(canvasIDs)

	err = models.SaveInvitation(invitation)
	if err != nil {
		log.Errorf("Error updating invitation %s with canvas IDs %v: %v", invitationID, canvasIDs, err)
		return nil, status.Error(codes.Internal, "failed to update invitation")
	}

	return &organizations.UpdateInvitationResponse{
		Invitation: serializeInvitation(invitation),
	}, nil
}
