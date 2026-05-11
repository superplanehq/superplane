package grpc

import (
	"context"

	"github.com/superplanehq/superplane/agent2/internal/anthropic"
	"github.com/superplanehq/superplane/agent2/internal/store"

	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ServiceConfig struct {
	Client        *anthropic.Client
	Store         *store.Store
	AgentID       string
	EnvironmentID string
}

type Service struct {
	internalpb.UnimplementedAgentsServer

	client        *anthropic.Client
	store         *store.Store
	agentID       string
	environmentID string
}

func NewService(cfg ServiceConfig) *Service {
	return &Service{
		client:        cfg.Client,
		store:         cfg.Store,
		agentID:       cfg.AgentID,
		environmentID: cfg.EnvironmentID,
	}
}

func (s *Service) CreateAgentChat(ctx context.Context, req *internalpb.CreateAgentChatRequest) (*internalpb.CreateAgentChatResponse, error) {
	if req.OrgId == "" || req.CanvasId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "org_id, canvas_id, and user_id are required")
	}

	// Create Anthropic session
	sessionReq := anthropic.CreateSessionRequest{
		Agent: s.agentID,
	}
	if s.environmentID != "" {
		sessionReq.EnvironmentID = s.environmentID
	}

	session, err := s.client.CreateSession(ctx, sessionReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create anthropic session: %v", err)
	}

	// Store mapping
	chat, err := s.store.CreateChat(ctx, req.OrgId, req.UserId, req.CanvasId, session.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store chat: %v", err)
	}

	return &internalpb.CreateAgentChatResponse{
		Chat: &internalpb.ChatInfo{
			Id:        chat.ID,
			CreatedAt: timestamppb.New(chat.CreatedAt),
		},
	}, nil
}

func (s *Service) ListAgentChats(ctx context.Context, req *internalpb.ListAgentChatsRequest) (*internalpb.ListAgentChatsResponse, error) {
	chats, err := s.store.ListChats(ctx, req.OrgId, req.UserId, req.CanvasId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list chats: %v", err)
	}

	var infos []*internalpb.ChatInfo
	for _, chat := range chats {
		infos = append(infos, &internalpb.ChatInfo{
			Id:             chat.ID,
			InitialMessage: chat.InitialMessage,
			CreatedAt:      timestamppb.New(chat.CreatedAt),
		})
	}

	return &internalpb.ListAgentChatsResponse{Chats: infos}, nil
}

func (s *Service) DescribeAgentChat(ctx context.Context, req *internalpb.DescribeAgentChatRequest) (*internalpb.DescribeAgentChatResponse, error) {
	chat, err := s.store.GetChat(ctx, req.OrgId, req.ChatId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "chat not found")
	}

	return &internalpb.DescribeAgentChatResponse{
		Chat: &internalpb.ChatInfo{
			Id:             chat.ID,
			InitialMessage: chat.InitialMessage,
			CreatedAt:      timestamppb.New(chat.CreatedAt),
		},
	}, nil
}

func (s *Service) ListAgentChatMessages(ctx context.Context, req *internalpb.ListAgentChatMessagesRequest) (*internalpb.ListAgentChatMessagesResponse, error) {
	chat, err := s.store.GetChat(ctx, req.OrgId, req.ChatId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "chat not found")
	}

	events, err := s.client.ListEvents(ctx, chat.AnthropicSessionID, 100)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}

	var messages []*internalpb.AgentChatMessage
	for _, event := range events.Data {
		msg := mapEventToMessage(event)
		if msg != nil {
			messages = append(messages, msg)
		}
	}

	return &internalpb.ListAgentChatMessagesResponse{Messages: messages}, nil
}

func (s *Service) DeleteAgentChat(ctx context.Context, req *internalpb.DeleteAgentChatRequest) (*internalpb.DeleteAgentChatResponse, error) {
	if err := s.store.DeleteChat(ctx, req.OrgId, req.ChatId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete chat: %v", err)
	}

	return &internalpb.DeleteAgentChatResponse{}, nil
}

func (s *Service) DescribeOrganizationAgentUsage(ctx context.Context, req *internalpb.DescribeOrganizationAgentUsageRequest) (*internalpb.DescribeOrganizationAgentUsageResponse, error) {
	// TODO: track usage per org
	return &internalpb.DescribeOrganizationAgentUsageResponse{
		Usage: &internalpb.ChatUsage{},
	}, nil
}

func mapEventToMessage(event anthropic.Event) *internalpb.AgentChatMessage {
	switch event.Type {
	case "user.message":
		text := ""
		for _, c := range event.Content {
			if c.Type == "text" {
				text += c.Text
			}
		}
		return &internalpb.AgentChatMessage{
			Id:      event.ID,
			Role:    "user",
			Content: text,
		}
	case "agent.message":
		text := ""
		for _, c := range event.Content {
			if c.Type == "text" {
				text += c.Text
			}
		}
		if text == "" {
			return nil
		}
		return &internalpb.AgentChatMessage{
			Id:      event.ID,
			Role:    "assistant",
			Content: text,
		}
	case "agent.tool_use":
		return &internalpb.AgentChatMessage{
			Id:         event.ID,
			Role:       "tool",
			Content:    event.Name,
			ToolCallId: event.ID,
			ToolStatus: "running",
		}
	case "agent.tool_result":
		return &internalpb.AgentChatMessage{
			Id:         event.ID,
			Role:       "tool",
			Content:    event.Name,
			ToolCallId: event.ID,
			ToolStatus: "completed",
		}
	default:
		return nil
	}
}
