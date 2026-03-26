package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/agentservice"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	agents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentsService struct {
	authService  authorization.Authorization
	agentService agentservice.Service
	jwtSigner    *jwt.Signer
}

func NewAgentsService(authService authorization.Authorization, agentService agentservice.Service, jwtSigner *jwt.Signer) *AgentsService {
	return &AgentsService{
		authService:  authService,
		agentService: agentService,
		jwtSigner:    jwtSigner,
	}
}

func (s *AgentsService) ListAgentChats(ctx context.Context, req *pb.ListAgentChatsRequest) (*pb.ListAgentChatsResponse, error) {
	organizationID, userID, err := s.getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.ListAgentChats(ctx, &internalpb.ListAgentChatsRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		UserId:   userID,
	})
	if err != nil {
		return nil, err
	}

	serialized := make([]*pb.AgentChatInfo, 0, len(response.Chats))
	for _, chat := range response.Chats {
		info, serializeErr := agents.SerializeAgentChatInfo(chat)
		if serializeErr != nil {
			return nil, serializeErr
		}
		serialized = append(serialized, info)
	}

	return &pb.ListAgentChatsResponse{
		Chats: serialized,
	}, nil
}

func (s *AgentsService) CreateAgentChat(ctx context.Context, req *pb.CreateAgentChatRequest) (*pb.CreateAgentChatResponse, error) {
	organizationID, userID, err := s.getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.CreateAgentChat(
		s.authService,
		s.jwtSigner,
		config.AgentHTTPURL(),
		userID,
		organizationID,
		req.CanvasId,
	)
}

func (s *AgentsService) DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	organizationID, userID, err := s.getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.DescribeAgentChat(ctx, &internalpb.DescribeAgentChatRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		UserId:   userID,
		ChatId:   req.ChatId,
	})
	if err != nil {
		return nil, err
	}

	info, err := agents.SerializeAgentChatInfo(response.Chat)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeAgentChatResponse{
		Chat: info,
	}, nil
}

func (s *AgentsService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	organizationID, userID, err := s.getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.ListAgentChatMessages(ctx, &internalpb.ListAgentChatMessagesRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		UserId:   userID,
		ChatId:   req.ChatId,
	})
	if err != nil {
		return nil, err
	}

	return &pb.ListAgentChatMessagesResponse{
		Messages: agents.SerializeAgentChatMessages(response.Messages),
	}, nil
}

func (s *AgentsService) ResumeAgentChat(ctx context.Context, req *pb.ResumeAgentChatRequest) (*pb.ResumeAgentChatResponse, error) {
	organizationID, userID, err := s.getUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.DescribeAgentChat(ctx, &internalpb.DescribeAgentChatRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		UserId:   userID,
		ChatId:   req.ChatId,
	})

	if err != nil {
		return nil, err
	}

	if response.GetChat() == nil {
		return nil, status.Error(codes.Internal, "agent service did not return a chat")
	}

	return agents.MintResumeAgentChatStreamResponse(
		s.jwtSigner,
		config.AgentHTTPURL(),
		userID,
		organizationID,
		req.CanvasId,
		response.GetChat().GetId(),
	)
}

func (s *AgentsService) getUserFromContext(ctx context.Context) (string, string, error) {
	organizationID, ok := ctx.Value(authorization.OrganizationContextKey).(string)
	if !ok || organizationID == "" {
		return "", "", status.Error(codes.Internal, "organization context is missing")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return "", "", err
	}

	return organizationID, userID, nil
}
