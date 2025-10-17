package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveInvitation(ctx context.Context, authService authorization.Authorization, orgID string, invitationID string) (*pb.RemoveInvitationResponse, error) {
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

	return &pb.RemoveInvitationResponse{}, nil
}
