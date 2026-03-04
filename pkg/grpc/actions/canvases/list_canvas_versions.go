package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListCanvasVersions(ctx context.Context, organizationID string, canvasID string) (*pb.ListCanvasVersionsResponse, error) {
	return ListCanvasVersionsPaginated(ctx, organizationID, canvasID, 0, nil)
}

func ListCanvasVersionsPaginated(
	ctx context.Context,
	organizationID string,
	canvasID string,
	limit uint32,
	before *timestamppb.Timestamp,
) (*pb.ListCanvasVersionsResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	userUUID := uuid.MustParse(userID)
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)

	var publishedVersions []models.CanvasVersion
	var publishedCount int64
	var draftVersion *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
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

		if beforeTime != nil {
			return nil
		}

		draft, draftErr := models.FindCanvasDraftInTransaction(tx, canvas.ID, userUUID)
		if draftErr != nil {
			if errors.Is(draftErr, gorm.ErrRecordNotFound) {
				return nil
			}
			return draftErr
		}

		version, versionErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, draft.VersionID)
		if versionErr != nil {
			return versionErr
		}
		draftVersion = version
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list canvas versions: %v", err)
	}

	protoVersions := make([]*pb.CanvasVersion, 0, len(publishedVersions)+1)
	for i := range publishedVersions {
		protoVersions = append(protoVersions, SerializeCanvasVersion(&publishedVersions[i], organizationID))
	}

	if draftVersion != nil {
		protoVersions = append(protoVersions, SerializeCanvasVersion(draftVersion, organizationID))
	}

	return &pb.ListCanvasVersionsResponse{
		Versions:      protoVersions,
		TotalCount:    uint32(publishedCount),
		HasNextPage:   hasNextPage(len(publishedVersions), int(limit), publishedCount),
		LastTimestamp: getLastCanvasVersionTimestamp(publishedVersions),
	}, nil
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
