package agents

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DefineAgentOutcome(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.DefineAgentOutcomeRequest) (*pb.DefineAgentOutcomeResponse, error) {
	org, user, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}
	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat id")
	}

	if req.Rubric == "" {
		return nil, status.Error(codes.InvalidArgument, "rubric is required")
	}

	maxIter := int(req.MaxIterations)
	if maxIter == 0 {
		maxIter = 3
	}

	if err := svc.DefineOutcome(ctx, org, user, chatID, req.Description, req.Rubric, maxIter); err != nil {
		translated := translateAgentServiceError(err, "failed to define agent outcome")
		if status.Code(translated) == codes.Internal {
			log.WithError(err).WithField("chat_id", chatID).Error("failed to define agent outcome")
		} else {
			log.WithError(err).WithField("chat_id", chatID).Warn("define agent outcome returned non-internal error")
		}
		return nil, translated
	}
	return &pb.DefineAgentOutcomeResponse{}, nil
}
