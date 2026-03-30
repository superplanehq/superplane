package canvases

import (
	"bytes"
	"context"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blobstorage"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errDuplicateBlobObjectKey = `duplicate key value violates unique constraint "idx_blobs_object_key"`

func StoreBlob(
	ctx context.Context,
	store blobstorage.BlobStorage,
	organizationID string,
	req *pb.StoreBlobRequest,
	userID string,
) (*pb.StoreBlobResponse, error) {
	scopeType, err := protoBlobScopeTypeToModel(req.ScopeType)
	if err != nil {
		return nil, err
	}

	trimmedPath := strings.TrimSpace(strings.TrimPrefix(req.Path, "/"))
	if trimmedPath == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}
	if req.ContentType != "" {
		pathExtension := strings.ToLower(filepath.Ext(trimmedPath))
		if pathExtension != "" {
			allowedExtensions, extErr := mime.ExtensionsByType(req.ContentType)
			if extErr == nil && len(allowedExtensions) > 0 {
				matchesContentType := false
				for _, extension := range allowedExtensions {
					if strings.EqualFold(extension, pathExtension) {
						matchesContentType = true
						break
					}
				}
				if !matchesContentType {
					return nil, status.Error(codes.InvalidArgument, "path extension does not match content type")
				}
			}
		}
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

	objectKey, err := scopeObjectKey(scopeType, canvasUUID, nodeID, executionUUID, trimmedPath)
	if err != nil {
		return nil, err
	}

	content := req.Content
	if len(content) == 0 {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	_, err = store.Put(ctx, blobstorage.PutInput{
		Key:         objectKey,
		Body:        bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: req.ContentType,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store blob content: %v", err)
	}

	now := time.Now()
	record := &models.Blob{
		OrganizationID: orgUUID,
		ScopeType:      scopeType,
		CanvasID:       canvasUUID,
		NodeID:         nodeID,
		ExecutionID:    executionUUID,
		Path:           trimmedPath,
		ObjectKey:      objectKey,
		SizeBytes:      int64(len(content)),
		ContentType:    req.ContentType,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	if userID != "" {
		if parsedUserID, parseErr := uuid.Parse(userID); parseErr == nil {
			record.CreatedByUserID = &parsedUserID
		}
	}

	if err := models.CreateBlob(record); err != nil {
		if strings.Contains(err.Error(), errDuplicateBlobObjectKey) {
			return nil, status.Error(codes.AlreadyExists, "a blob with this path already exists in this scope")
		}
		_ = store.Delete(ctx, objectKey)
		return nil, status.Errorf(codes.Internal, "failed to store blob metadata: %v", err)
	}

	return &pb.StoreBlobResponse{
		Blob: toProtoBlob(record),
	}, nil
}
