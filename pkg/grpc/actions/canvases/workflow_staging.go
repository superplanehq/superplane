package canvases

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

const staleStagingMessage = "Main branch has been updated since you last edited. Discard your changes and start again."

type canvasStagingContext struct {
	canvas      *models.Canvas
	liveVersion *models.CanvasVersion
	userID      uuid.UUID
	rows        []models.WorkflowStaging
}

func publishStagingUpdated(canvasID uuid.UUID) {
	if err := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), "").PublishStagingUpdated(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}
}

func loadCanvasStagingContext(
	ctx context.Context,
	organizationID string,
	canvasID string,
) (*canvasStagingContext, error) {
	db := database.DB(ctx)
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvasInTransaction(db, organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load live version")
	}

	userUUID := uuid.MustParse(userID)
	rows, err := models.ListWorkflowStagingForUserInTransaction(db, canvas.ID, userUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	return &canvasStagingContext{
		canvas:      canvas,
		liveVersion: liveVersion,
		userID:      userUUID,
		rows:        rows,
	}, nil
}

func ensureStagingNotStale(staging *canvasStagingContext) error {
	if staging == nil || len(staging.rows) == 0 {
		return nil
	}

	baseVersionID := models.StagingBaseVersionID(staging.rows)
	if staging.canvas.LiveVersionID == nil || baseVersionID != *staging.canvas.LiveVersionID {
		return grpcerrors.FailedPrecondition(nil, staleStagingMessage)
	}

	return nil
}

func buildStagingSummary(canvas *models.Canvas, rows []models.WorkflowStaging) *pb.StagingSummary {
	state := &pb.StagingSummary{}
	if len(rows) == 0 {
		return state
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Path)
	}

	base := models.StagingBaseVersionID(rows).String()
	state.HasStaging = true
	state.StagedPaths = paths
	state.BaseVersionId = &base
	if canvas != nil && canvas.LiveVersionID != nil {
		state.Stale = *canvas.LiveVersionID != models.StagingBaseVersionID(rows)
	}

	return state
}

func stagingSummaryForCanvas(canvas *models.Canvas, userID uuid.UUID) (*pb.StagingSummary, []models.WorkflowStaging, error) {
	rows, err := models.ListWorkflowStagingForUser(nil, canvas.ID, userID)
	if err != nil {
		return nil, nil, grpcerrors.Internal(err, "failed to load staging")
	}
	return buildStagingSummary(canvas, rows), rows, nil
}

func stagingBaseVersionID(canvas *models.Canvas, rows []models.WorkflowStaging) uuid.UUID {
	if len(rows) > 0 {
		return models.StagingBaseVersionID(rows)
	}
	if canvas != nil && canvas.LiveVersionID != nil {
		return *canvas.LiveVersionID
	}
	return uuid.Nil
}

func effectiveSpecYAML(
	canvas *models.Canvas,
	version *models.CanvasVersion,
	organizationID string,
	rows []models.WorkflowStaging,
	path string,
) (string, error) {
	for _, row := range rows {
		if row.Path != path {
			continue
		}
		if row.Deleted {
			return "", nil
		}
		return row.Content, nil
	}

	switch path {
	case CanvasYAMLRepositoryPath:
		return canvasYAMLFromVersion(canvas, version, organizationID)
	case ConsoleYAMLRepositoryPath:
		return consoleYAMLFromVersion(canvas, version)
	default:
		return "", grpcerrors.InvalidArgument(nil, fmt.Sprintf("unsupported repository spec file %q", path))
	}
}

func PutCanvasStaging(
	ctx context.Context,
	organizationID string,
	canvasID string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.StagingSummary, error) {
	staging, err := loadCanvasStagingContext(ctx, organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	if err := ensureStagingNotStale(staging); err != nil {
		return nil, err
	}

	baseVersionID := stagingBaseVersionID(staging.canvas, staging.rows)
	organizationUUID := staging.canvas.OrganizationID

	for _, operation := range operations {
		if operation == nil {
			continue
		}

		normalized := normalizeRepositoryFilePath(operation.GetPath())
		if normalized == "" {
			return nil, grpcerrors.InvalidArgument(nil, "file path is required")
		}
		if normalized == gitprovider.ReservedSuperPlanePath ||
			strings.HasPrefix(normalized, gitprovider.ReservedSuperPlanePath+"/") {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("path %q is reserved for SuperPlane", operation.GetPath()))
		}

		if operation.GetDelete() {
			if err := models.MarkWorkflowStagingPathDeleted(
				nil,
				staging.canvas.ID,
				staging.userID,
				baseVersionID,
				organizationUUID,
				normalized,
				&staging.userID,
			); err != nil {
				return nil, grpcerrors.Internal(err, "failed to stage deletion")
			}
			continue
		}

		if _, err := models.UpsertWorkflowStagingPath(
			nil,
			staging.canvas.ID,
			staging.userID,
			baseVersionID,
			organizationUUID,
			normalized,
			string(operation.GetContent()),
			&staging.userID,
		); err != nil {
			return nil, grpcerrors.Internal(err, "failed to stage")
		}
	}

	rows, err := models.ListWorkflowStagingForUser(nil, staging.canvas.ID, staging.userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	publishStagingUpdated(staging.canvas.ID)

	return buildStagingSummary(staging.canvas, rows), nil
}

func GetCanvasStaging(
	ctx context.Context,
	organizationID string,
	canvasID string,
) (*pb.StagingSummary, error) {
	staging, err := loadCanvasStagingContext(ctx, organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	return buildStagingSummary(staging.canvas, staging.rows), nil
}

func DeleteCanvasStaging(
	ctx context.Context,
	organizationID string,
	canvasID string,
	paths []string,
) (*pb.StagingSummary, error) {
	staging, err := loadCanvasStagingContext(ctx, organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	if err := models.DiscardWorkflowStagingForUser(nil, staging.canvas.ID, staging.userID, paths); err != nil {
		return nil, grpcerrors.Internal(err, "failed to discard staging")
	}

	state, _, err := stagingSummaryForCanvas(staging.canvas, staging.userID)
	if err != nil {
		return nil, err
	}

	publishStagingUpdated(staging.canvas.ID)

	return state, nil
}

func ReadStagedRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	path string,
) (content string, found bool, deleted bool, err error) {
	staging, loadErr := loadCanvasStagingContext(ctx, organizationID, canvasID)
	if loadErr != nil {
		return "", false, false, loadErr
	}

	normalized := normalizeRepositoryFilePath(path)
	for _, row := range staging.rows {
		if row.Path != normalized {
			continue
		}
		if row.Deleted {
			return "", true, true, nil
		}
		return row.Content, true, false, nil
	}

	return "", false, false, nil
}

func ReadStagedRepositorySpecFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	version *models.CanvasVersion,
	path string,
) (string, error) {
	staging, err := loadCanvasStagingContext(ctx, organizationID, canvasID)
	if err != nil {
		return "", err
	}

	return effectiveSpecYAML(staging.canvas, version, organizationID, staging.rows, path)
}
