package agents

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	maxChatImages            = 8
	maxChatImagePayloadBytes = 2_500_000
)

func SendAgentChatMessage(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.SendAgentChatMessageRequest) (*pb.SendAgentChatMessageResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat id")
	}
	images, err := parseChatImages(req.Images)
	if err != nil {
		return nil, err
	}
	if req.Content == "" && len(images) == 0 {
		return nil, status.Error(codes.InvalidArgument, "content or an image is required")
	}

	persisted, err := svc.SendMessage(ctx, org, user, chatID, req.Content, images, agentModeFromProto(req.Mode))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "agent chat not found")
		}
		if errors.Is(err, agentservice.ErrSessionBusy) {
			return nil, status.Error(codes.FailedPrecondition, "agent is still processing the previous turn")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to send agent chat message")
		return nil, status.Error(codes.Internal, "failed to send agent chat message")
	}
	return &pb.SendAgentChatMessageResponse{Message: serializeMessage(persisted)}, nil
}

func parseChatImages(images []*pb.AgentChatImage) ([]agentservice.MessageImage, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if len(images) > maxChatImages {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d images are allowed per message", maxChatImages)
	}

	out := make([]agentservice.MessageImage, 0, len(images))
	total := 0
	for _, image := range images {
		mediaType, ok := chatImageMediaTypeToContentType(image.MediaType)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported image media type: %s", image.MediaType)
		}
		if len(image.Data) == 0 {
			return nil, status.Error(codes.InvalidArgument, "image data is empty")
		}
		total += len(image.Data)
		if total > maxChatImagePayloadBytes {
			return nil, status.Errorf(codes.InvalidArgument, "images exceed the %d byte limit per message", maxChatImagePayloadBytes)
		}
		out = append(out, agentservice.MessageImage{
			MediaType: mediaType,
			Data:      base64.StdEncoding.EncodeToString(image.Data),
		})
	}
	return out, nil
}

func chatImageMediaTypeToContentType(mediaType pb.AgentChatImageMediaType) (string, bool) {
	switch mediaType {
	case pb.AgentChatImageMediaType_MEDIA_TYPE_PNG:
		return "image/png", true
	case pb.AgentChatImageMediaType_MEDIA_TYPE_JPEG:
		return "image/jpeg", true
	case pb.AgentChatImageMediaType_MEDIA_TYPE_GIF:
		return "image/gif", true
	case pb.AgentChatImageMediaType_MEDIA_TYPE_WEBP:
		return "image/webp", true
	default:
		return "", false
	}
}
