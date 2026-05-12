package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentsService struct {
	service *agents.Service
}

func NewAgentsService(service *agents.Service) *AgentsService {
	return &AgentsService{service: service}
}

func (s *AgentsService) CreateAgentChat(ctx context.Context, req *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error) {
	orgID, userID, err := s.extractIDs(ctx)
	if err != nil {
		return nil, err
	}
	return s.service.CreateAgentChat(ctx, orgID, userID, req.CanvasId)
}

func (s *AgentsService) ResumeAgentChat(ctx context.Context, req *pb.ResumeAgentChatRequest) (*pb.ResumeAgentChatResponse, error) {
	orgID, userID, err := s.extractIDs(ctx)
	if err != nil {
		return nil, err
	}
	return s.service.ResumeAgentChat(ctx, orgID, userID, req.CanvasId)
}

func (s *AgentsService) DeleteAgentChat(ctx context.Context, req *pb.DeleteAgentChatRequest) (*pb.DeleteAgentChatResponse, error) {
	orgID, userID, err := s.extractIDs(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.service.DeleteAgentChat(orgID, userID, req.CanvasId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete chat: %v", err)
	}
	return &pb.DeleteAgentChatResponse{}, nil
}

func (s *AgentsService) ListAgentChats(ctx context.Context, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	// With single-session model, just return the one session if it exists
	orgID, userID, err := s.extractIDs(ctx)
	if err != nil {
		return nil, err
	}

	_ = orgID
	_ = userID
	_ = req

	return &pb.ListAgentChatsResponse{}, nil
}

func (s *AgentsService) DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	return &pb.DescribeAgentChatResponse{}, nil
}

func (s *AgentsService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	orgID, userID, err := s.extractIDs(ctx)
	if err != nil {
		return nil, err
	}
	return s.service.ListAgentChatMessages(orgID, userID, req.CanvasId)
}

func (s *AgentsService) extractIDs(ctx context.Context) (string, string, error) {
	orgID, ok := ctx.Value(authorization.OrganizationContextKey).(string)
	if !ok || orgID == "" {
		return "", "", status.Error(codes.Unauthenticated, "organization not found")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return "", "", err
	}

	return orgID, userID, nil
}
