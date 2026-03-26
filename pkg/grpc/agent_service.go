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
	agentURL := config.AgentHTTPURL()
	if agentURL == "" {
		return nil, status.Error(codes.Unavailable, "agent HTTP URL not configured")
	}

	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.CreateAgentChat(
		s.authService,
		s.jwtSigner,
		agentURL,
		userID,
		organizationID,
		req.CanvasId,
	)
}

func (s *AgentsService) DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *AgentsService) ListAgentChats(ctx context.Context, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *AgentsService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *AgentsService) ResumeAgentChat(ctx context.Context, req *pb.ResumeAgentChatRequest) (*pb.ResumeAgentChatResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
