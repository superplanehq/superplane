package me

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
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
	if _, err := models.ActiveCodexMember(database.DB(ctx), orgID, userID); err != nil {
		return nil, grpcerrors.PermissionDenied(err, "Codex requires active organization membership and the Codex feature")
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
			view.Limits = append(view.Limits, &pb.CodexUsageLimit{
				LimitId:   limit.LimitID,
				LimitName: limit.LimitName,
				PlanType:  limit.PlanType,
				Primary:   serializeCodexUsageWindow(limit.Primary),
				Secondary: serializeCodexUsageWindow(limit.Secondary),
			})
		}
		response.Connections = append(response.Connections, view)
	}
	return response, nil
}

func serializeCodexUsageWindow(window *models.CodexUsageWindow) *pb.CodexUsageWindow {
	if window == nil || window.UsedPercent == nil {
		return nil
	}
	view := &pb.CodexUsageWindow{
		UsedPercent:        *window.UsedPercent,
		WindowDurationMins: window.WindowDurationMins,
	}
	if window.ResetsAt != nil {
		view.ResetsAt = timestamppb.New(time.Unix(*window.ResetsAt, 0))
	}
	return view
}
