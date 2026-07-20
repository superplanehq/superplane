package agents

import (
	"context"
	"errors"
	"github.com/superplanehq/superplane/pkg/grpc/errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"gorm.io/gorm"
)

func DefineAgentOutcome(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.DefineAgentOutcomeRequest) (*pb.DefineAgentOutcomeResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid chat id")
	}

	if req.Rubric == "" {
		return nil, grpcerrors.InvalidArgument(nil, "rubric is required")
	}

	maxIter := int(req.MaxIterations)
	if maxIter == 0 {
		maxIter = 3
	}

	if err := svc.DefineOutcome(ctx, org, user, chatID, req.Description, req.Rubric, maxIter); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "agent chat not found")
		}
		if errors.Is(err, agentservice.ErrSessionBusy) {
			return nil, grpcerrors.FailedPrecondition(nil, "agent is still processing the previous turn")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to define agent outcome")
		return nil, grpcerrors.Internal(err, "failed to define agent outcome")
	}
	return &pb.DefineAgentOutcomeResponse{}, nil
}
