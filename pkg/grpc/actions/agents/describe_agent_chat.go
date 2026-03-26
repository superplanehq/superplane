package agents

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func DescribeAgentChat(
	ctx context.Context,
	agentURL string,
	orgID string,
	userID string,
	canvasID string,
	chatID string,
) (*pb.DescribeAgentChatResponse, error) {
	conn, err := grpc.NewClient(agentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to create agent GRPC client")
	}

	client := internalpb.NewAgentsClient(conn)
	response, err := client.DescribeAgentChat(ctx, &internalpb.DescribeAgentChatRequest{
		OrgId:    orgID,
		UserId:   userID,
		CanvasId: canvasID,
		ChatId:   chatID,
	})

	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to describe agent chat")
	}

	if response.Chat == nil {
		return nil, status.Error(codes.NotFound, "agent chat not found")
	}

	chat := &pb.AgentChatInfo{
		Id:             response.Chat.Id,
		InitialMessage: response.Chat.InitialMessage,
	}

	if response.Chat.CreatedAt != nil {
		chat.CreatedAt = timestamppb.New(response.Chat.CreatedAt.AsTime())
	}

	return &pb.DescribeAgentChatResponse{
		Chat: chat,
	}, nil
}
