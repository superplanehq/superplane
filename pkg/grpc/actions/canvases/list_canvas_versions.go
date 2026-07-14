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
	organizationID string,
	canvasID string,
	limit uint32,
	before *timestamppb.Timestamp,
) (*pb.ListCanvasVersionsResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	_, err = loadCanvas(ctx, database.DB(ctx), orgUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	limit = getCanvasVersionLimit(limit)
	beforeTime := getBefore(before)

	versions, count, err := listCanvasVersionHistory(ctx, canvasUUID, int(limit), beforeTime)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list canvas versions")
	}

	protoVersions := serializeCanvasVersions(ctx, versions, organizationID)

	return &pb.ListCanvasVersionsResponse{
		Versions:      protoVersions,
		TotalCount:    uint32(count),
		HasNextPage:   hasNextPage(len(versions), int(limit), count),
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

func getLastCanvasVersionTimestamp(versions []models.CanvasVersion) *timestamppb.Timestamp {
	if len(versions) == 0 {
		return nil
	}

	lastVersion := versions[len(versions)-1]
	if lastVersion.CreatedAt == nil {
		return nil
	}

	return timestamppb.New(*lastVersion.CreatedAt)
}

func listCanvasVersionHistory(ctx context.Context, canvasUUID uuid.UUID, limit int, beforeTime *time.Time) (versions []models.CanvasVersion, count int64, err error) {
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
