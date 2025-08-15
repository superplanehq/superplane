package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListInvitations(ctx context.Context, orgID string) (*pb.ListInvitationsResponse, error) {
	invitations, err := models.ListInvitationsInState(orgID, models.InvitationStatePending)
	if err != nil {
		log.Errorf("error listing invitations for %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "error listing invitations")
	}

	return &pb.ListInvitationsResponse{
		Invitations: serializeInvitations(invitations),
	}, nil
}
