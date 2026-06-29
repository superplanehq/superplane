package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
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
		return nil, grpcerrors.InvalidArgument(nil, "invalid chat id")
	}

	if _, err := svc.GetSession(org, user, chatID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "agent chat not found")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to load agent chat")
		return nil, grpcerrors.Internal(err, "failed to load agent chat")
	}

	var beforeID uuid.UUID
	if req.BeforeId != "" {
		beforeID, err = uuid.Parse(req.BeforeId)
		if err != nil {
			return nil, grpcerrors.InvalidArgument(nil, "invalid before_id")
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = defaultMessagePageLimit
	}

	messages, err := svc.ListMessages(chatID, beforeID, limit)
	if err != nil {
		log.WithError(err).WithField("chat_id", chatID).Error("failed to list agent chat messages")
		return nil, grpcerrors.Internal(err, "failed to list messages")
	}

	out := make([]*pb.AgentChatMessage, 0, len(messages))
	for i := range messages {
		out = append(out, serializeMessage(&messages[i]))
	}
	return &pb.ListAgentChatMessagesResponse{Messages: out, HasMore: len(messages) == limit}, nil
}
