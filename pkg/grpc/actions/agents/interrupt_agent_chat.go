package agents

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func InterruptAgentChat(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.InterruptAgentChatRequest) (*pb.InterruptAgentChatResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat id")
	}

	if err := svc.InterruptSession(ctx, org, user, chatID); err != nil {
		translated := translateAgentServiceError(err, "failed to interrupt agent chat")
		if status.Code(translated) == codes.Internal {
			log.WithError(err).WithField("chat_id", chatID).Error("failed to interrupt agent chat")
		} else {
			log.WithError(err).WithField("chat_id", chatID).Warn("interrupt agent chat returned non-internal error")
		}
		return nil, translated
	}
	return &pb.InterruptAgentChatResponse{}, nil
}
