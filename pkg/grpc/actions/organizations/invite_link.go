package organizations

import (
	"context"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

const inviteLinksUnavailableMessage = "Invite links are no longer available. Use email invitations."

func GetInviteLink(_ context.Context, _ string) (*pb.GetInviteLinkResponse, error) {
	return nil, grpcerrors.FailedPrecondition(nil, inviteLinksUnavailableMessage)
}

func UpdateInviteLink(_ context.Context, _ string, _ bool) (*pb.UpdateInviteLinkResponse, error) {
	return nil, grpcerrors.FailedPrecondition(nil, inviteLinksUnavailableMessage)
}

func ResetInviteLink(_ context.Context, _ string) (*pb.ResetInviteLinkResponse, error) {
	return nil, grpcerrors.FailedPrecondition(nil, inviteLinksUnavailableMessage)
}
