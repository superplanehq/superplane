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
)

func DeleteAgentChat(
	ctx context.Context,
	agentURL string,
	orgID string,
	userID string,
	canvasID string,
	chatID string,
) (*pb.DeleteAgentChatResponse, error) {
	conn, err := grpc.NewClient(agentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to create agent GRPC client")
	}
	defer closeAgentConnection(conn)

	client := internalpb.NewAgentsClient(conn)
	_, err = client.DeleteAgentChat(ctx, &internalpb.DeleteAgentChatRequest{
		OrgId:    orgID,
		UserId:   userID,
		CanvasId: canvasID,
		ChatId:   chatID,
	})

	if err != nil {
		log.WithError(err).Errorf("failed to delete agent chat %s for org %s, user %s", chatID, orgID, userID)
		return nil, status.Error(codes.Unavailable, "failed to delete agent chat")
	}

	return &pb.DeleteAgentChatResponse{}, nil
}
