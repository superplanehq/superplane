package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListBlobs(
	_ context.Context,
	organizationID string,
	req *pb.ListBlobsRequest,
) (*pb.ListBlobsResponse, error) {
	scopeType, err := protoBlobScopeTypeToModel(req.ScopeType)
	if err != nil {
		return nil, err
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}

	var canvasUUID *uuid.UUID
	if req.CanvasId != "" {
		parsed, parseErr := uuid.Parse(req.CanvasId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
		}
		canvasUUID = &parsed
	}

	var executionUUID *uuid.UUID
	if req.ExecutionId != "" {
		parsed, parseErr := uuid.Parse(req.ExecutionId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid execution_id")
		}
		executionUUID = &parsed
	}

	var nodeID *string
	if req.NodeId != "" {
		nodeValue := req.NodeId
		nodeID = &nodeValue
	}

	limit := getBlobListLimit(req.Limit)
	before := parseBeforeTimestamp(req.Before)

	rows, err := models.ListBlobsByScope(orgUUID, scopeType, canvasUUID, nodeID, executionUUID, limit+1, before)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list blobs: %v", err)
	}

	hasNextPage := blobHasNextPage(len(rows), limit+1)
	if hasNextPage {
		rows = rows[:limit]
	}

	blobs := make([]*pb.Blob, 0, len(rows))
	for idx := range rows {
		blobs = append(blobs, toProtoBlob(&rows[idx]))
	}

	var lastTimestamp *timestamppb.Timestamp
	if len(rows) > 0 && rows[len(rows)-1].CreatedAt != nil {
		lastTimestamp = timestamppb.New(*rows[len(rows)-1].CreatedAt)
	}

	return &pb.ListBlobsResponse{
		Blobs:         blobs,
		HasNextPage:   hasNextPage,
		LastTimestamp: lastTimestamp,
	}, nil
}
