package agents

import (
	"context"
	"errors"
	"github.com/superplanehq/superplane/pkg/grpc/errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"gorm.io/gorm"
)

func InterruptAgentChat(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.InterruptAgentChatRequest) (*pb.InterruptAgentChatResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid chat id")
	}

	if err := svc.InterruptSession(ctx, org, user, chatID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "agent chat not found")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to interrupt agent chat")
		return nil, grpcerrors.Internal(err, "failed to interrupt agent chat")
	}
	return &pb.InterruptAgentChatResponse{}, nil
}
