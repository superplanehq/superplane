package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const agentTokenTTL = 1 * time.Hour

// Service implements the agent business logic.
type Service struct {
	Client    *Client
	Store     *Store
	JWTSigner *jwt.Signer
	BaseURL   string // SuperPlane API base URL for CLI config
}

func NewService(client *Client, store *Store, jwtSigner *jwt.Signer, baseURL string) *Service {
	return &Service{
		Client:    client,
		Store:     store,
		JWTSigner: jwtSigner,
		BaseURL:   baseURL,
	}
}

// GenerateAgentToken creates a short-lived scoped token for the agent CLI.
func (s *Service) GenerateAgentToken(orgID, userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(agentTokenTTL)

	token, err := s.JWTSigner.GenerateScopedToken(jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   orgID,
		Purpose: "agent",
		Scopes:  []string{"canvases:read", "canvases:write", "integrations:read", "components:read"},
	}, agentTokenTTL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate agent token: %w", err)
	}

	return token, expiresAt, nil
}

// CreateAgentChat returns the existing session or creates a new one.
func (s *Service) CreateAgentChat(ctx context.Context, orgID, userID, canvasID string) (*pb.CreateAgentChatResponse, error) {
	// Check if session already exists
	existing, err := s.Store.FindSession(orgID, userID, canvasID)
	if err == nil {
		// Refresh token if expired or missing
		if err := s.refreshTokenIfNeeded(existing); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to refresh agent token: %v", err)
		}

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
	stored, err := s.Store.CreateSession(orgID, userID, canvasID, session.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store session: %v", err)
	}

	// Generate and store scoped token
	token, expiresAt, err := s.GenerateAgentToken(orgID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate agent token: %v", err)
	}

	if err := s.Store.UpdateAPIToken(stored.ID, token, expiresAt); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store agent token: %v", err)
	}

	return &pb.CreateAgentChatResponse{
		Url: fmt.Sprintf("/api/v1/agents/chats/%s/stream", canvasID),
	}, nil
}

// refreshTokenIfNeeded regenerates the token if it's expired or missing.
func (s *Service) refreshTokenIfNeeded(session *ChatSession) error {
	needsRefresh := session.APIToken == nil ||
		*session.APIToken == "" ||
		session.APITokenExpiresAt == nil ||
		time.Now().After(*session.APITokenExpiresAt)

	if !needsRefresh {
		return nil
	}

	token, expiresAt, err := s.GenerateAgentToken(session.OrganizationID, session.UserID)
	if err != nil {
		return err
	}

	return s.Store.UpdateAPIToken(session.ID, token, expiresAt)
}

// ResumeAgentChat returns the stream URL for an existing session.
func (s *Service) ResumeAgentChat(ctx context.Context, orgID, userID, canvasID string) (*pb.ResumeAgentChatResponse, error) {
	session, err := s.Store.FindSession(orgID, userID, canvasID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "no session found for this canvas")
	}

	// Refresh token on resume
	if err := s.refreshTokenIfNeeded(session); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to refresh agent token: %v", err)
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
