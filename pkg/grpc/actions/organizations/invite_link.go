package organizations

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetInviteLink(ctx context.Context, orgID string) (*pb.GetInviteLinkResponse, error) {
	db := database.DB(ctx)
	orgUUID, parseErr := uuid.Parse(orgID)
	if parseErr != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization id")
	}

	inviteLink, err := models.FindOrCreateInviteLink(db, orgUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get invite link")
	}

	return &pb.GetInviteLinkResponse{
		InviteLink: serializeInviteLink(inviteLink),
	}, nil
}

func UpdateInviteLink(ctx context.Context, orgID string, enabled bool) (*pb.UpdateInviteLinkResponse, error) {
	db := database.DB(ctx)
	inviteLink, err := models.FindInviteLinkByOrganizationID(db, orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "invite link not found")
	}

	inviteLink.Enabled = enabled
	inviteLink.UpdatedAt = time.Now()
	if err := models.SaveInviteLink(db, inviteLink); err != nil {
		return nil, grpcerrors.Internal(err, "failed to update invite link")
	}

	return &pb.UpdateInviteLinkResponse{
		InviteLink: serializeInviteLink(inviteLink),
	}, nil
}

func ResetInviteLink(ctx context.Context, orgID string) (*pb.ResetInviteLinkResponse, error) {
	db := database.DB(ctx)
	inviteLink, err := models.FindInviteLinkByOrganizationID(db, orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "invite link not found")
	}

	inviteLink.Token = uuid.New()
	inviteLink.UpdatedAt = time.Now()
	if err := models.SaveInviteLink(db, inviteLink); err != nil {
		return nil, grpcerrors.Internal(err, "failed to reset invite link")
	}

	return &pb.ResetInviteLinkResponse{
		InviteLink: serializeInviteLink(inviteLink),
	}, nil
}

func serializeInviteLink(inviteLink *models.OrganizationInviteLink) *pb.InviteLink {
	return &pb.InviteLink{
		Id:             inviteLink.ID.String(),
		OrganizationId: inviteLink.OrganizationID.String(),
		Token:          inviteLink.Token.String(),
		Enabled:        inviteLink.Enabled,
		CreatedAt:      timestamppb.New(inviteLink.CreatedAt),
		UpdatedAt:      timestamppb.New(inviteLink.UpdatedAt),
	}
}
