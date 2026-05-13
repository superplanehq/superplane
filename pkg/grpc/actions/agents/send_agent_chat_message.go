package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// SendAgentChatMessage returns the persisted user message. The assistant
// reply streams back asynchronously over the chat's websocket topic.
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

	persisted, err := svc.SendMessage(ctx, org, user, chatID, req.Content)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "agent chat not found")
		}
		if errors.Is(err, agents.ErrSessionAlreadyTerminated) {
			return nil, status.Error(codes.FailedPrecondition, "agent chat is archived")
		}
		return nil, status.Errorf(codes.Internal, "failed to send agent chat message: %v", err)
	}
	return &pb.SendAgentChatMessageResponse{Message: serializeMessage(persisted)}, nil
}
