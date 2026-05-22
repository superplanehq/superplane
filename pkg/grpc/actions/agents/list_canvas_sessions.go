package agents

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvasSessions(ctx context.Context, svc AgentsService, orgID, userID string, req *pb.ListCanvasSessionsRequest) (*pb.ListCanvasSessionsResponse, error) {
	org, _, err := parseOrgUser(orgID, userID)
	if err != nil {
		return nil, err
	}

	canvasID, err := uuid.Parse(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas id")
	}

	if err := ensureCanvas(org, canvasID); err != nil {
		return nil, err
	}

	entries, err := svc.ListCanvasSessions(org, canvasID)
	if err != nil {
		log.WithError(err).Error("failed to list canvas sessions")
		return nil, status.Error(codes.Internal, "failed to list canvas sessions")
	}

	sessions := make([]*pb.CanvasSessionInfo, len(entries))
	for i, e := range entries {
		var lastActivity *timestamppb.Timestamp
		if e.LastActiveAt != nil {
			lastActivity = timestamppb.New(*e.LastActiveAt)
		}
		sessions[i] = &pb.CanvasSessionInfo{
			Id:             e.SessionID,
			UserId:         e.UserID,
			UserName:       e.UserName,
			Status:         e.Status,
			LastActivityAt: lastActivity,
		}
	}

	return &pb.ListCanvasSessionsResponse{Sessions: sessions}, nil
}
