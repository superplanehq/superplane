package agents

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListAgentChats(_ context.Context, svc AgentsService, orgID, userID string, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	org, user, canvas, err := parseCanvasScope(orgID, userID, req.CanvasId)
	if err != nil {
		return nil, err
	}
	if err := ensureCanvas(org, canvas); err != nil {
		return nil, err
	}

	sessions, err := svc.ListSessions(org, user, canvas)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list agent chats: %v", err)
	}

	out := make([]*pb.AgentChatInfo, 0, len(sessions))
	for i := range sessions {
		out = append(out, serializeChat(&sessions[i]))
	}
	return &pb.ListAgentChatsResponse{Chats: out}, nil
}
