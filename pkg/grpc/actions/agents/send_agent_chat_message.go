package agents

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	maxChatImages = 8

	// maxChatImagePayloadBytes caps the combined decoded image bytes per message
	// so the base64-encoded request stays under the gRPC server's 4 MiB receive
	// limit.
	maxChatImagePayloadBytes = 3 * 1000 * 1000
)

var allowedChatImageMediaTypes = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

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
		if !slices.Contains(allowedChatImageMediaTypes, image.MediaType) {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported image media type: %q", image.MediaType)
		}
		decoded, err := base64.StdEncoding.DecodeString(image.Data)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "image data must be valid base64")
		}
		if len(decoded) == 0 {
			return nil, status.Error(codes.InvalidArgument, "image data is empty")
		}
		total += len(decoded)
		if total > maxChatImagePayloadBytes {
			return nil, status.Errorf(codes.InvalidArgument, "images exceed the %d byte limit per message", maxChatImagePayloadBytes)
		}
		out = append(out, agentservice.MessageImage{MediaType: image.MediaType, Data: image.Data})
	}
	return out, nil
}
