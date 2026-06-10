package agents

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SendAgentChatMessage(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.SendAgentChatMessageRequest) (*pb.SendAgentChatMessageResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat id")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	persisted, err := svc.SendMessage(ctx, org, user, chatID, req.Content, agentModeFromProto(req.Mode))
	if err != nil {
		translated := translateAgentServiceError(err, "failed to send agent chat message")
		if status.Code(translated) == codes.Internal {
			log.WithError(err).WithField("chat_id", chatID).Error("failed to send agent chat message")
		} else {
			log.WithError(err).WithField("chat_id", chatID).Warn("send agent chat message returned non-internal error")
		}
		return nil, translated
	}
	return &pb.SendAgentChatMessageResponse{Message: serializeMessage(persisted)}, nil
}
