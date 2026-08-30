package me

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeUserAPIToken(token *models.UserAPIToken) *pb.UserAPIToken {
	out := &pb.UserAPIToken{
		Id:        token.ID.String(),
		Name:      token.Name,
		CreatedAt: timestamppb.New(token.CreatedAt),
	}

	if token.LastUsedAt != nil {
		out.LastUsedAt = timestamppb.New(*token.LastUsedAt)
	}

	return out
}
