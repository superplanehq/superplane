package agents

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateAgentChat(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error) {
	org, user, canvas, err := parseCanvasScope(orgID, userID, req.CanvasId)
	if err != nil {
		return nil, err
	}
	if err := ensureCanvas(org, canvas); err != nil {
		return nil, err
	}

	session, err := svc.CreateSession(ctx, org, user, canvas)
	if err != nil {
		if errors.Is(err, agents.ErrSessionForbidden) {
			return nil, status.Error(codes.PermissionDenied, "agent chat is not allowed")
		}
		return nil, status.Errorf(codes.Internal, "failed to create agent chat: %v", err)
	}
	return &pb.CreateAgentChatResponse{Chat: serializeChat(session)}, nil
}
