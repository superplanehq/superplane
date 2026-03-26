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

func ListAgentChats(ctx context.Context, agentURL string, orgID string, userID string, canvasID string) (*pb.ListAgentChatsResponse, error) {
	conn, err := grpc.NewClient(agentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to create agent GRPC client")
	}

	client := internalpb.NewAgentsClient(conn)
	response, err := client.ListAgentChats(ctx, &internalpb.ListAgentChatsRequest{
		OrgId:    orgID,
		UserId:   userID,
		CanvasId: canvasID,
	})

	if err != nil {
		log.WithError(err).Errorf("failed to list agent chats for org %s, user %s, canvas %s", orgID, userID, canvasID)
		return nil, status.Error(codes.Unavailable, "failed to list agent chats")
	}

	return &pb.ListAgentChatsResponse{
		Chats: serializeAgentChats(response.Chats),
	}, nil
}

func serializeAgentChats(in []*internalpb.ChatInfo) []*pb.AgentChatInfo {
	out := make([]*pb.AgentChatInfo, 0, len(in))
	for _, chat := range in {
		if chat == nil {
			continue
		}

		chat := &pb.AgentChatInfo{
			Id:             chat.Id,
			InitialMessage: chat.InitialMessage,
		}

		if chat.CreatedAt != nil {
			chat.CreatedAt = timestamppb.New(chat.CreatedAt.AsTime())
		}

		out = append(out, chat)
	}

	return out
}
