package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeAgentChat(_ context.Context, svc AgentsService, orgID, userID string, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat id")
	}

	session, err := svc.GetSession(org, user, chatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "agent chat not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to load agent chat: %v", err)
	}
	return &pb.DescribeAgentChatResponse{Chat: serializeChat(session)}, nil
}
