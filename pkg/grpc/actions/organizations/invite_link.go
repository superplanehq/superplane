package organizations

import (
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func GetInviteLink(orgID string) (*pb.GetInviteLinkResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}

	inviteLink, err := models.FindInviteLinkByOrganizationID(orgID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("Error fetching invite link for organization %s: %v", orgID, err)
			return nil, status.Error(codes.Internal, "failed to fetch invite link")
		}

		// Seed the invite link on demand. CreateInviteLink is implemented as
		// an idempotent upsert so concurrent requests for the same
		// organization don't fail with a unique-violation when both hit
		// this branch.
		inviteLink, err = models.CreateInviteLink(orgUUID)
		if err != nil {
			log.Errorf("Error creating invite link for organization %s: %v", orgID, err)
			return nil, status.Error(codes.Internal, "failed to create invite link")
		}
	}

	return &pb.GetInviteLinkResponse{
		InviteLink: serializeInviteLink(inviteLink),
	}, nil
}

func UpdateInviteLink(orgID string, enabled bool) (*pb.UpdateInviteLinkResponse, error) {
	inviteLink, err := models.FindInviteLinkByOrganizationID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invite link not found")
	}

	inviteLink.Enabled = enabled
	inviteLink.UpdatedAt = time.Now()
	if err := models.SaveInviteLink(inviteLink); err != nil {
		log.Errorf("Error updating invite link for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to update invite link")
	}

	return &pb.UpdateInviteLinkResponse{
		InviteLink: serializeInviteLink(inviteLink),
	}, nil
}

func ResetInviteLink(orgID string) (*pb.ResetInviteLinkResponse, error) {
	inviteLink, err := models.FindInviteLinkByOrganizationID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invite link not found")
	}

	inviteLink.Token = uuid.New()
	inviteLink.UpdatedAt = time.Now()
	if err := models.SaveInviteLink(inviteLink); err != nil {
		log.Errorf("Error resetting invite link for organization %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "failed to reset invite link")
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
