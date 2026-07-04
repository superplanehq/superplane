// Package agents implements the gRPC handlers for the public Agents service.
package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type AgentsService interface {
	EnsureSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error)
	GetSession(organizationID, userID, sessionID uuid.UUID) (*models.AgentSession, error)
	ListMessages(sessionID, beforeID uuid.UUID, limit int) ([]models.AgentSessionMessage, error)
	SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string, images []agentservice.MessageImage, mode ...string) (*models.AgentSessionMessage, error)
	InterruptSession(ctx context.Context, organizationID, userID, sessionID uuid.UUID) error
	DefineOutcome(ctx context.Context, organizationID, userID, sessionID uuid.UUID, description, rubric string, maxIterations int) error
}

func agentModeFromProto(mode pb.AgentMode) string {
	switch mode {
	case pb.AgentMode_MODE_BUILDER:
		return string(agentservice.ModeBuilder)
	case pb.AgentMode_MODE_ARCHITECT:
		return string(agentservice.ModeOperator)
	default:
		return string(agentservice.ModeOperator)
	}
}

func parseOrgUser(orgID, userID string) (org, user uuid.UUID, err error) {
	org, err = uuid.Parse(orgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, grpcerrors.InvalidArgument(err, "invalid organization")
	}
	user, err = uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, grpcerrors.InvalidArgument(err, "invalid user")
	}
	return org, user, nil
}

func ensureCanvas(orgID, canvasID uuid.UUID) error {
	if _, err := models.FindCanvas(orgID, canvasID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return grpcerrors.NotFound(err, "canvas not found")
		}
		return grpcerrors.Internal(err, "failed to load canvas")
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
		Images:     serializeImages(message.Images),
	}
	if message.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*message.CreatedAt)
	}
	return out
}

func serializeImages(images []models.AgentSessionImage) []*pb.AgentChatImage {
	if len(images) == 0 {
		return nil
	}
	out := make([]*pb.AgentChatImage, 0, len(images))
	for _, image := range images {
		mediaType := contentTypeToChatImageMediaType(image.MediaType)
		if mediaType == pb.AgentChatImageMediaType_MEDIA_TYPE_UNSPECIFIED {
			continue
		}
		out = append(out, &pb.AgentChatImage{MediaType: mediaType})
	}
	return out
}

func contentTypeToChatImageMediaType(mediaType string) pb.AgentChatImageMediaType {
	switch mediaType {
	case "image/png":
		return pb.AgentChatImageMediaType_MEDIA_TYPE_PNG
	case "image/jpeg":
		return pb.AgentChatImageMediaType_MEDIA_TYPE_JPEG
	case "image/gif":
		return pb.AgentChatImageMediaType_MEDIA_TYPE_GIF
	case "image/webp":
		return pb.AgentChatImageMediaType_MEDIA_TYPE_WEBP
	default:
		return pb.AgentChatImageMediaType_MEDIA_TYPE_UNSPECIFIED
	}
}
