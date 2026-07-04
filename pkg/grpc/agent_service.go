package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	agentsActions "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentsService struct {
	pb.UnimplementedAgentsServer
	service agentsActions.AgentsService
}

func NewAgentsService(service agentsActions.AgentsService) *AgentsService {
	return &AgentsService{service: service}
}

func (s *AgentsService) ensureEnabled() error {
	if s.service == nil {
		return status.Error(codes.Unavailable, "agents are not enabled on this installation")
	}
	return nil
}

func (s *AgentsService) requestContext(ctx context.Context) (orgID, userID string, err error) {
	if err := s.ensureEnabled(); err != nil {
		return "", "", err
	}
	orgIDVal, _ := ctx.Value(authorization.OrganizationContextKey).(string)
	if orgIDVal == "" {
		return "", "", status.Error(codes.Unauthenticated, "missing organization")
	}
	userID, err = userIDFromContext(ctx)
	if err != nil {
		return "", "", err
	}
	return orgIDVal, userID, nil
}

func (s *AgentsService) GetCanvasAgentChat(ctx context.Context, req *pb.GetCanvasAgentChatRequest) (*pb.GetCanvasAgentChatResponse, error) {
	orgID, userID, err := s.requestContext(ctx)
	if err != nil {
		return nil, err
	}
	return agentsActions.GetCanvasAgentChat(ctx, s.service, orgID, userID, req)
}

func (s *AgentsService) ResetCanvasAgentChat(ctx context.Context, req *pb.ResetCanvasAgentChatRequest) (*pb.ResetCanvasAgentChatResponse, error) {
	orgID, userID, err := s.requestContext(ctx)
	if err != nil {
		return nil, err
	}
	return agentsActions.ResetCanvasAgentChat(ctx, s.service, orgID, userID, req)
}

func (s *AgentsService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	orgID, userID, err := s.requestContext(ctx)
	if err != nil {
		return nil, err
	}
	return agentsActions.ListAgentChatMessages(ctx, s.service, orgID, userID, req)
}

func (s *AgentsService) SendAgentChatMessage(ctx context.Context, req *pb.SendAgentChatMessageRequest) (*pb.SendAgentChatMessageResponse, error) {
	orgID, userID, err := s.requestContext(ctx)
	if err != nil {
		return nil, err
	}
	return agentsActions.SendAgentChatMessage(ctx, s.service, orgID, userID, req)
}

func (s *AgentsService) InterruptAgentChat(ctx context.Context, req *pb.InterruptAgentChatRequest) (*pb.InterruptAgentChatResponse, error) {
	orgID, userID, err := s.requestContext(ctx)
	if err != nil {
		return nil, err
	}
	return agentsActions.InterruptAgentChat(ctx, s.service, orgID, userID, req)
}

func (s *AgentsService) DefineAgentOutcome(ctx context.Context, req *pb.DefineAgentOutcomeRequest) (*pb.DefineAgentOutcomeResponse, error) {
	orgID, userID, err := s.requestContext(ctx)
	if err != nil {
		return nil, err
	}
	return agentsActions.DefineAgentOutcome(ctx, s.service, orgID, userID, req)
}
