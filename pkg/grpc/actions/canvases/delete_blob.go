package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blobstorage"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteBlob(
	ctx context.Context,
	store blobstorage.BlobStorage,
	organizationID string,
	id string,
) error {
	blobID, err := uuid.Parse(id)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid blob id")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid organization id")
	}

	row, err := models.FindBlob(blobID)
	if err != nil {
		return status.Errorf(codes.NotFound, "blob not found")
	}
	if row.OrganizationID != orgUUID {
		return status.Error(codes.NotFound, "blob not found")
	}

	if err := store.Delete(ctx, row.ObjectKey); err != nil && !errors.Is(err, blobstorage.ErrBlobNotFound) {
		return status.Errorf(codes.Internal, "failed to delete blob content: %v", err)
	}

	if err := models.DeleteBlob(blobID); err != nil {
		return status.Errorf(codes.Internal, "failed to delete blob metadata: %v", err)
	}

	return nil
}
