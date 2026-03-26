package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	agents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentsService struct {
	authService authorization.Authorization
	jwtSigner   *jwt.Signer
}

func NewAgentsService(authService authorization.Authorization, jwtSigner *jwt.Signer) *AgentsService {
	return &AgentsService{
		authService: authService,
		jwtSigner:   jwtSigner,
	}
}

func (s *AgentsService) CreateAgentChat(ctx context.Context, req *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error) {
	agentPublicURL := config.AgentHTTPURL()
	if agentPublicURL == "" {
		return nil, status.Error(codes.Unavailable, "agent HTTP URL not configured")
	}

	agentInternalURL := config.AgentGRPCURL()
	if agentInternalURL == "" {
		return nil, status.Error(codes.Unavailable, "agent GRPC URL not configured")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.CreateAgentChat(
		ctx,
		s.authService,
		s.jwtSigner,
		agentInternalURL,
		agentPublicURL,
		userID,
		organizationID,
		req.CanvasId,
	)
}

func (s *AgentsService) ResumeAgentChat(ctx context.Context, req *pb.ResumeAgentChatRequest) (*pb.ResumeAgentChatResponse, error) {
	agentPublicURL := config.AgentHTTPURL()
	if agentPublicURL == "" {
		return nil, status.Error(codes.Unavailable, "agent HTTP URL not configured")
	}

	agentInternalURL := config.AgentGRPCURL()
	if agentInternalURL == "" {
		return nil, status.Error(codes.Unavailable, "agent GRPC URL not configured")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	url := config.AgentGRPCURL()
	if url == "" {
		return nil, status.Error(codes.Unavailable, "agent GRPC URL not configured")
	}

	return agents.ResumeAgentChat(
		ctx,
		s.authService,
		s.jwtSigner,
		agentInternalURL,
		agentPublicURL,
		organizationID,
		userID,
		req.CanvasId,
		req.ChatId,
	)
}

func (s *AgentsService) DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	url := config.AgentGRPCURL()
	if url == "" {
		return nil, status.Error(codes.Unavailable, "agent GRPC URL not configured")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.DescribeAgentChat(ctx, url, organizationID, userID, req.CanvasId, req.ChatId)
}

func (s *AgentsService) ListAgentChats(ctx context.Context, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	url := config.AgentGRPCURL()
	if url == "" {
		return nil, status.Error(codes.Unavailable, "agent GRPC URL not configured")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.ListAgentChats(ctx, url, organizationID, userID, req.CanvasId)
}

func (s *AgentsService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	url := config.AgentGRPCURL()
	if url == "" {
		return nil, status.Error(codes.Unavailable, "agent GRPC URL not configured")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.ListAgentChatMessages(ctx, url, organizationID, userID, req.CanvasId, req.ChatId)
}
