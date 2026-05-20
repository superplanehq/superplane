package agents

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "agent chat not found")
		}
		log.WithError(err).WithField("chat_id", chatID).Error("failed to define agent outcome")
		return nil, status.Error(codes.Internal, "failed to define agent outcome")
	}
	return &pb.DefineAgentOutcomeResponse{}, nil
}
