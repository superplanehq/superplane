package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
)

func GetCanvasAgentChat(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.GetCanvasAgentChatRequest) (*pb.GetCanvasAgentChatResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	canvas, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid canvas id")
	}
	if err := ensureCanvas(org, canvas); err != nil {
		return nil, err
	}

	session, err := svc.EnsureSession(ctx, org, user, canvas)
	if err != nil {
		if errors.Is(err, agents.ErrSessionForbidden) {
			return nil, grpcerrors.PermissionDenied(nil, "agent chat is not allowed")
		}
		log.WithError(err).WithField("canvas_id", canvas).Error("failed to ensure agent chat")
		return nil, grpcerrors.Internal(err, "failed to load agent chat")
	}
	return &pb.GetCanvasAgentChatResponse{Chat: serializeChat(session)}, nil
}
