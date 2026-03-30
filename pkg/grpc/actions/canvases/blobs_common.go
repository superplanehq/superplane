package canvases

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func protoBlobScopeTypeToModel(scope pb.Blob_ScopeType) (string, error) {
	switch scope {
	case pb.Blob_SCOPE_TYPE_ORGANIZATION:
		return models.BlobScopeOrganization, nil
	case pb.Blob_SCOPE_TYPE_CANVAS:
		return models.BlobScopeCanvas, nil
	case pb.Blob_SCOPE_TYPE_NODE:
		return models.BlobScopeNode, nil
	case pb.Blob_SCOPE_TYPE_EXECUTION:
		return models.BlobScopeExecution, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid blob scope type: %v", scope)
	}
}

func modelBlobScopeTypeToProto(scope string) pb.Blob_ScopeType {
	switch scope {
	case models.BlobScopeOrganization:
		return pb.Blob_SCOPE_TYPE_ORGANIZATION
	case models.BlobScopeCanvas:
		return pb.Blob_SCOPE_TYPE_CANVAS
	case models.BlobScopeNode:
		return pb.Blob_SCOPE_TYPE_NODE
	case models.BlobScopeExecution:
		return pb.Blob_SCOPE_TYPE_EXECUTION
	default:
		return pb.Blob_SCOPE_TYPE_UNSPECIFIED
	}
}

func scopeObjectKey(scopeType string, canvasID *uuid.UUID, nodeID *string, executionID *uuid.UUID, path string) (string, error) {
	switch scopeType {
	case models.BlobScopeOrganization:
		return fmt.Sprintf("blobs/organization/%s", path), nil
	case models.BlobScopeCanvas:
		if canvasID == nil {
			return "", status.Error(codes.InvalidArgument, "canvas_id is required for canvas scope")
		}
		return fmt.Sprintf("blobs/canvas/%s/%s", canvasID.String(), path), nil
	case models.BlobScopeNode:
		if canvasID == nil || nodeID == nil || *nodeID == "" {
			return "", status.Error(codes.InvalidArgument, "canvas_id and node_id are required for node scope")
		}
		return fmt.Sprintf("blobs/node/%s/%s/%s", canvasID.String(), *nodeID, path), nil
	case models.BlobScopeExecution:
		if executionID == nil {
			return "", status.Error(codes.InvalidArgument, "execution_id is required for execution scope")
		}
		return fmt.Sprintf("blobs/execution/%s/%s", executionID.String(), path), nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid blob scope type: %s", scopeType)
	}
}

func toProtoBlob(blob *models.Blob) *pb.Blob {
	if blob == nil {
		return nil
	}

	result := &pb.Blob{
		Id:             blob.ID.String(),
		OrganizationId: blob.OrganizationID.String(),
		ScopeType:      modelBlobScopeTypeToProto(blob.ScopeType),
		Path:           blob.Path,
		ObjectKey:      blob.ObjectKey,
		SizeBytes:      blob.SizeBytes,
		ContentType:    blob.ContentType,
	}

	if blob.CanvasID != nil {
		result.CanvasId = blob.CanvasID.String()
	}
	if blob.NodeID != nil {
		result.NodeId = *blob.NodeID
	}
	if blob.ExecutionID != nil {
		result.ExecutionId = blob.ExecutionID.String()
	}
	if blob.CreatedAt != nil {
		result.CreatedAt = timestamppb.New(*blob.CreatedAt)
	}

	return result
}

func getBlobListLimit(limit uint32) int {
	if limit == 0 || limit > 100 {
		return 100
	}
	return int(limit)
}

func blobHasNextPage(numResults int, limit int) bool {
	return numResults >= limit
}

func parseBeforeTimestamp(before *timestamppb.Timestamp) *time.Time {
	if before == nil {
		return nil
	}

	value := before.AsTime()
	return &value
}
