package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const MaxCanvasVersionLimit = 50

func ListCanvasVersions(ctx context.Context, organizationID string, canvasID string) (*pb.ListCanvasVersionsResponse, error) {
	return ListCanvasVersionsPaginated(ctx, organizationID, canvasID, 0, nil, pb.CanvasVersion_STATE_UNSPECIFIED)
}

func ListCanvasVersionsPaginated(
	ctx context.Context,
	organizationID string,
	canvasID string,
	limit uint32,
	before *timestamppb.Timestamp,
	state pb.CanvasVersion_State,
) (*pb.ListCanvasVersionsResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	canvas, err := findCanvas(ctx, orgUUID, canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if state == pb.CanvasVersion_STATE_DRAFT {
		return listDraftCanvasVersions(ctx, organizationID, canvas, uuid.MustParse(userID), limit, before)
	}

	limit = getCanvasVersionLimit(limit)
	beforeTime := getBefore(before)

	var publishedVersions []models.CanvasVersion
	var publishedCount int64
	err = telemetry.RunSpan(ctx, "canvases.list_published_versions", func(ctx context.Context) error {
		return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
			versions, versionsErr := models.ListPublishedCanvasVersionsInTransaction(tx, canvas.ID, int(limit), beforeTime)
			if versionsErr != nil {
				return versionsErr
			}
			publishedVersions = versions

			count, countErr := models.CountPublishedCanvasVersionsInTransaction(tx, canvas.ID)
			if countErr != nil {
				return countErr
			}
			publishedCount = count

			return nil
		})
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list canvas versions: %v", err)
	}

	protoVersions := make([]*pb.CanvasVersion, 0, len(publishedVersions))
	for i := range publishedVersions {
		protoVersions = append(protoVersions, SerializeCanvasVersion(&publishedVersions[i], organizationID))
	}

	return &pb.ListCanvasVersionsResponse{
		Versions:      protoVersions,
		TotalCount:    uint32(publishedCount),
		HasNextPage:   hasNextPage(len(publishedVersions), int(limit), publishedCount),
		LastTimestamp: getLastCanvasVersionTimestamp(publishedVersions),
	}, nil
}

func listDraftCanvasVersions(
	ctx context.Context,
	organizationID string,
	canvas *models.Canvas,
	ownerID uuid.UUID,
	limit uint32,
	before *timestamppb.Timestamp,
) (*pb.ListCanvasVersionsResponse, error) {
	limit = getCanvasVersionLimit(limit)
	beforeTime := getBefore(before)

	var draftVersions []models.CanvasVersion
	var draftCount int64
	err := telemetry.RunSpan(ctx, "canvases.list_draft_versions", func(ctx context.Context) error {
		return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
			versions, versionsErr := models.ListDraftBranchesForCanvasInTransaction(tx, canvas.ID, ownerID, int(limit), beforeTime)
			if versionsErr != nil {
				return versionsErr
			}
			draftVersions = versions

			count, countErr := models.CountDraftBranchesForCanvasInTransaction(tx, canvas.ID, ownerID)
			if countErr != nil {
				return countErr
			}
			draftCount = count

			return nil
		})
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list canvas versions: %v", err)
	}

	protoVersions := make([]*pb.CanvasVersion, 0, len(draftVersions))
	for i := range draftVersions {
		protoVersions = append(protoVersions, SerializeCanvasVersion(&draftVersions[i], organizationID))
	}

	return &pb.ListCanvasVersionsResponse{
		Versions:      protoVersions,
		TotalCount:    uint32(draftCount),
		HasNextPage:   hasNextPage(len(draftVersions), int(limit), draftCount),
		LastTimestamp: getLastDraftCanvasVersionTimestamp(draftVersions),
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
	if lastVersion.PublishedAt == nil {
		return nil
	}

	return timestamppb.New(*lastVersion.PublishedAt)
}

func getLastDraftCanvasVersionTimestamp(versions []models.CanvasVersion) *timestamppb.Timestamp {
	if len(versions) == 0 {
		return nil
	}

	lastVersion := versions[len(versions)-1]
	if lastVersion.UpdatedAt == nil {
		return nil
	}

	return timestamppb.New(*lastVersion.UpdatedAt)
}
