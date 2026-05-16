// Package agents implements the gRPC handlers for the public Agents service.
package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type AgentsService interface {
	EnsureSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error)
	GetSession(organizationID, userID, sessionID uuid.UUID) (*models.AgentSession, error)
	ListMessages(sessionID, beforeID uuid.UUID, limit int) ([]models.AgentSessionMessage, error)
	SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string, mode ...string) (*models.AgentSessionMessage, error)
	InterruptSession(ctx context.Context, organizationID, userID, sessionID uuid.UUID) error
}

func parseOrgUser(orgID, userID string) (org, user uuid.UUID, err error) {
	org, err = uuid.Parse(orgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.Internal, "invalid organization")
	}
	user, err = uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.Internal, "invalid user")
	}
	return org, user, nil
}

func ensureCanvas(orgID, canvasID uuid.UUID) error {
	if _, err := models.FindCanvas(orgID, canvasID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "canvas not found")
		}
		return status.Error(codes.Internal, "failed to load canvas")
	}
	return nil
}

func serializeChat(session *models.AgentSession) *pb.AgentChatInfo {
	info := &pb.AgentChatInfo{
		Id:       session.ID.String(),
		CanvasId: session.CanvasID.String(),
		Provider: session.Provider,
		Status:   session.Status,
	}
	if session.CreatedAt != nil {
		info.CreatedAt = timestamppb.New(*session.CreatedAt)
	}
	if session.UpdatedAt != nil {
		info.UpdatedAt = timestamppb.New(*session.UpdatedAt)
	}
	return info
}

func serializeMessage(message *models.AgentSessionMessage) *pb.AgentChatMessage {
	out := &pb.AgentChatMessage{
		Id:         message.ID.String(),
		Role:       message.Role,
		Content:    message.Content,
		ToolCallId: message.ToolCallID,
		ToolName:   message.ToolName,
		ToolStatus: message.ToolStatus,
	}
	if message.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*message.CreatedAt)
	}
	return out
}
