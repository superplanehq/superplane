package organizations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func GetInviteLink(ctx context.Context, orgID string) (*pb.GetInviteLinkResponse, error) {
	db := database.DB(ctx)
	inviteLink, err := models.FindInviteLinkByOrganizationID(db, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			orgUUID, parseErr := uuid.Parse(orgID)
			if parseErr != nil {
				return nil, grpcerrors.InvalidArgument(nil, "invalid organization id")
			}

			inviteLink, err = models.CreateInviteLink(orgUUID)
			if err != nil {
				return nil, grpcerrors.Internal(err, "failed to create invite link")
			}
		} else {
			return nil, grpcerrors.Internal(err, "failed to fetch invite link")
		}
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
	if err := models.SaveInviteLink(inviteLink); err != nil {
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
	if err := models.SaveInviteLink(inviteLink); err != nil {
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
