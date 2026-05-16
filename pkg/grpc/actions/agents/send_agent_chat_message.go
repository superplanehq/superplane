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

	persisted, err := svc.SendMessage(ctx, org, user, chatID, req.Content, req.Mode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "agent chat not found")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to send agent chat message")
		return nil, status.Error(codes.Internal, "failed to send agent chat message")
	}
	return &pb.SendAgentChatMessageResponse{Message: serializeMessage(persisted)}, nil
}
