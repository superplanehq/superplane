package me

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCodexUsage(ctx context.Context) (*pb.ListCodexUsageResponse, error) {
	orgID, err := currentOrganizationID(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := models.ActiveCodexConnectionMember(database.DB(ctx), orgID, userID); err != nil {
		return nil, grpcerrors.PermissionDenied(err, "Codex requires active organization membership")
	}
	connections, err := models.ListOwnedFactoryCodexConnections(database.DB(ctx), orgID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load Codex usage")
	}
	response := &pb.ListCodexUsageResponse{}
	for _, connection := range connections {
		view := &pb.CodexUsageConnection{
			Id:           connection.ID.String(),
			AccountLabel: connection.AccountLabel,
			Status:       connection.Status,
		}
		if connection.UsageObservedAt != nil {
			view.ObservedAt = timestamppb.New(*connection.UsageObservedAt)
		}
		for _, limit := range connection.UsageLimits {
			view.Limits = append(view.Limits, actions.SerializeCodexUsageLimit(limit))
		}
		response.Connections = append(response.Connections, view)
	}
	return response, nil
}
