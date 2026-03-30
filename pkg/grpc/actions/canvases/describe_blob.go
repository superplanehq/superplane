package canvases

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blobstorage"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeBlob(
	ctx context.Context,
	store blobstorage.BlobStorage,
	organizationID string,
	id string,
) (*pb.DescribeBlobResponse, error) {
	blobID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid blob id")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization id")
	}

	row, err := models.FindBlob(blobID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "blob not found")
	}
	if row.OrganizationID != orgUUID {
		return nil, status.Error(codes.NotFound, "blob not found")
	}

	content, err := store.Get(ctx, row.ObjectKey)
	if err != nil {
		if errors.Is(err, blobstorage.ErrBlobNotFound) {
			return nil, status.Error(codes.NotFound, "blob content not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to read blob content: %v", err)
	}
	defer content.Body.Close()

	data, err := io.ReadAll(content.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read blob body: %v", err)
	}

	return &pb.DescribeBlobResponse{
		Blob:    toProtoBlob(row),
		Content: data,
	}, nil
}
