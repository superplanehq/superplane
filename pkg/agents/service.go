package agents

import (
	"context"
	"fmt"

	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the agent business logic.
type Service struct {
	Client *Client
	Store  *Store
}

func NewService(client *Client, store *Store) *Service {
	return &Service{Client: client, Store: store}
}

// CreateAgentChat returns the existing session or creates a new one.
func (s *Service) CreateAgentChat(ctx context.Context, orgID, userID, canvasID string) (*pb.CreateAgentChatResponse, error) {
	// Check if session already exists
	existing, err := s.Store.FindSession(orgID, userID, canvasID)
	if err == nil {
		_ = existing
		return &pb.CreateAgentChatResponse{
			Url: fmt.Sprintf("/api/v1/agents/chats/%s/stream", canvasID),
		}, nil
	}

	// Create new Anthropic session
	session, err := s.Client.CreateSession(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create agent session: %v", err)
	}

	// Store it
	_, err = s.Store.CreateSession(orgID, userID, canvasID, session.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store session: %v", err)
	}

	return &pb.CreateAgentChatResponse{
		Url: fmt.Sprintf("/api/v1/agents/chats/%s/stream", canvasID),
	}, nil
}

// ResumeAgentChat returns the stream URL for an existing session.
func (s *Service) ResumeAgentChat(ctx context.Context, orgID, userID, canvasID string) (*pb.ResumeAgentChatResponse, error) {
	_, err := s.Store.FindSession(orgID, userID, canvasID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "no session found for this canvas")
	}

	return &pb.ResumeAgentChatResponse{
		Url: fmt.Sprintf("/api/v1/agents/chats/%s/stream", canvasID),
	}, nil
}

// DeleteAgentChat removes the session.
func (s *Service) DeleteAgentChat(orgID, userID, canvasID string) error {
	return s.Store.DeleteSession(orgID, userID, canvasID)
}

// ListAgentChatMessages returns stored messages.
func (s *Service) ListAgentChatMessages(orgID, userID, canvasID string) (*pb.ListAgentChatMessagesResponse, error) {
	session, err := s.Store.FindSession(orgID, userID, canvasID)
	if err != nil {
		return &pb.ListAgentChatMessagesResponse{}, nil
	}

	messages, err := s.Store.ListMessages(session.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list messages: %v", err)
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
