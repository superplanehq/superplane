package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	orgID, userID, err := s.extractIDs(ctx)
	if err != nil {
		return nil, err
	}

	// Single session per canvas/user — return it if it exists
	session, err := s.service.Store.FindSession(orgID, userID, req.CanvasId)
	if err != nil {
		return &pb.ListAgentChatsResponse{}, nil
	}

	// Get the first user message as initial_message
	var initialMessage string
	msgs, _ := s.service.Store.ListMessages(session.ID)
	for _, m := range msgs {
		if m.Role == "user" {
			initialMessage = m.Content
			break
		}
	}

	return &pb.ListAgentChatsResponse{
		Chats: []*pb.AgentChatInfo{{
			Id:             session.ID,
			InitialMessage: initialMessage,
			CreatedAt:      timestamppb.New(session.CreatedAt),
		}},
	}, nil
}

func (s *AgentsService) DescribeAgentChat(ctx context.Context, req *pb.DescribeAgentChatRequest) (*pb.DescribeAgentChatResponse, error) {
	return &pb.DescribeAgentChatResponse{}, nil
}

func (s *AgentsService) ListAgentChatMessages(ctx context.Context, req *pb.ListAgentChatMessagesRequest) (*pb.ListAgentChatMessagesResponse, error) {
	// chat_id is the session ID from our DB
	messages, err := s.service.Store.ListMessages(req.ChatId)
	if err != nil {
		return &pb.ListAgentChatMessagesResponse{}, nil
	}

	var pbMessages []*pb.AgentChatMessage
	for _, m := range messages {
		pbMessages = append(pbMessages, &pb.AgentChatMessage{
			Id:         m.ID,
			Role:       m.Role,
			Content:    m.Content,
			ToolCallId: m.ToolCallID,
			ToolStatus: m.ToolStatus,
			CreatedAt:  timestamppb.New(m.CreatedAt),
		})
	}

	return &pb.ListAgentChatMessagesResponse{Messages: pbMessages}, nil
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
