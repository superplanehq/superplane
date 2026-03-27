package agents

import (
	"context"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListAgentChatMessages(ctx context.Context, agentURL string, orgID string, userID string, canvasID string, chatID string) (*pb.ListAgentChatMessagesResponse, error) {
	conn, err := grpc.NewClient(agentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to create agent GRPC client")
	}

	client := internalpb.NewAgentsClient(conn)
	response, err := client.ListAgentChatMessages(ctx, &internalpb.ListAgentChatMessagesRequest{
		OrgId:    orgID,
		UserId:   userID,
		CanvasId: canvasID,
		ChatId:   chatID,
	})

	if err != nil {
		log.WithError(err).Errorf("failed to list agent chat messages for org %s, user %s, canvas %s, chat %s", orgID, userID, canvasID, chatID)
		return nil, status.Error(codes.Unavailable, "failed to list agent chat messages")
	}

	return &pb.ListAgentChatMessagesResponse{
		Messages: serializeAgentChatMessages(response.Messages),
	}, nil
}

func serializeAgentChatMessages(in []*internalpb.AgentChatMessage) []*pb.AgentChatMessage {
	out := make([]*pb.AgentChatMessage, 0, len(in))
	for _, message := range in {
		if message == nil {
			continue
		}

		m := &pb.AgentChatMessage{
			Id:         message.Id,
			Role:       message.Role,
			Content:    message.Content,
			ToolCallId: message.ToolCallId,
			ToolStatus: message.ToolStatus,
		}

		if message.CreatedAt != nil {
			m.CreatedAt = timestamppb.New(message.CreatedAt.AsTime())
		}

		out = append(out, m)
	}

	return out
}
