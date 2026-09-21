package canvases

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const MaxCanvasVersionLimit = 50

func ListCanvasVersionsPaginated(
	ctx context.Context,
	db *gorm.DB,
	canvas *models.Canvas,
	limit uint32,
	before *timestamppb.Timestamp,
) (*pb.ListCanvasVersionsResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	limit = getCanvasVersionLimit(limit)
	beforeTime := getBefore(before)

	versions, count, err := listCanvasVersionHistory(ctx, canvas.ID, int(limit)+1, beforeTime)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list canvas versions")
	}

	versions, hasNext := trimPage(versions, int(limit))

	protoVersions := serializeCanvasVersionMetadataList(ctx, versions, canvas.OrganizationID.String())

	return &pb.ListCanvasVersionsResponse{
		Versions:      protoVersions,
		TotalCount:    uint32(count),
		HasNextPage:   hasNext,
		LastTimestamp: getLastCanvasVersionTimestamp(versions),
	}, nil
}

func getCanvasVersionLimit(limit uint32) uint32 {
	if limit <= 0 {
		return DefaultLimit
	}

	if limit > MaxCanvasVersionLimit {
		return MaxCanvasVersionLimit
	}

	return limit
}

func getLastCanvasVersionTimestamp(versions []models.CanvasVersionMetadata) *timestamppb.Timestamp {
	if len(versions) == 0 {
		return nil
	}

	lastVersion := versions[len(versions)-1]
	if lastVersion.CreatedAt == nil {
		return nil
	}

	return timestamppb.New(*lastVersion.CreatedAt)
}

func listCanvasVersionHistory(ctx context.Context, canvasUUID uuid.UUID, limit int, beforeTime *time.Time) (versions []models.CanvasVersionMetadata, count int64, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.list_version_history")
	defer done(&err)

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		versions, txErr = models.ListCanvasVersionHistoryInTransaction(tx, canvasUUID, limit, beforeTime)
		if txErr != nil {
			return txErr
		}

		count, txErr = models.CountCanvasVersionsInTransaction(tx, canvasUUID)
		return txErr
	})

	return versions, count, err
}
