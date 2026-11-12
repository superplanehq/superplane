package me

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
)

func ListModelStatus(ctx context.Context, req *pb.ListModelStatusRequest) (*pb.ListModelStatusResponse, error) {
	if _, err := currentUserID(ctx); err != nil {
		return nil, err
	}

	window := ""
	if req != nil {
		window = req.GetWindow()
	}
	normalized, duration, ok := models.HostedModelCallOutcomeWindowDuration(window)
	if !ok {
		return nil, grpcerrors.InvalidArgument(nil, "window must be 5m, 15m, 1h, or 24h")
	}

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cells, err := models.ListHostedModelStatusCells(database.DB(queryCtx), time.Now().Add(-duration))
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load model status")
	}

	response := &pb.ListModelStatusResponse{Window: normalized}
	for _, cell := range cells {
		// Scores only. Call counts would show every customer's volume.
		view := &pb.ModelStatusCell{
			Provider: cell.Provider,
			Model:    cell.Model,
		}
		if cell.Score != nil {
			score := int32(*cell.Score)
			view.Score = &score
		}
		response.Cells = append(response.Cells, view)
	}
	return response, nil
}
