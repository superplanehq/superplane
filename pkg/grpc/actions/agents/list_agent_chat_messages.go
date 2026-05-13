package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const defaultMessagePageLimit = 50

func ListAgentChatMessages(_ context.Context, svc AgentsService, orgID, userID string, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat id")
	}

	if _, err := svc.GetSession(org, user, chatID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "agent chat not found")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to load agent chat")
		return nil, status.Error(codes.Internal, "failed to load agent chat")
	}

	var beforeID uuid.UUID
	if req.BeforeId != "" {
		beforeID, err = uuid.Parse(req.BeforeId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid before_id")
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = defaultMessagePageLimit
	}

	messages, err := svc.ListMessages(chatID, beforeID, limit)
	if err != nil {
		log.WithError(err).WithField("chat_id", chatID).Error("failed to list agent chat messages")
		return nil, status.Error(codes.Internal, "failed to list messages")
	}

	out := make([]*pb.AgentChatMessage, 0, len(messages))
	for i := range messages {
		out = append(out, serializeMessage(&messages[i]))
	}
	return &pb.ListAgentChatMessagesResponse{Messages: out, HasMore: len(messages) == limit}, nil
}
