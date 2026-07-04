package canvases

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
	"strings"
)

// publishStagingUpdated notifies other tabs/replicas that a draft version's
// staging layer changed so they can refetch staged caches. Failures are logged
// but never block the staging write.
func publishStagingUpdated(canvasID, versionID uuid.UUID) {
	if err := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), versionID.String()).PublishStagingUpdated(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}
}

// loadOwnedDraftVersion resolves the canvas and draft version for a staging
// write/commit/discard, enforcing that the caller owns the registered draft.
func loadOwnedDraftVersion(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*models.Canvas, *models.CanvasVersion, uuid.UUID, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, nil, uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, nil, uuid.Nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, uuid.Nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, nil, uuid.Nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, uuid.Nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, nil, uuid.Nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	version, err := models.FindCanvasVersion(canvas.ID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, uuid.Nil, grpcerrors.NotFound(err, "version not found")
		}
		return nil, nil, uuid.Nil, grpcerrors.Internal(err, "failed to load version")
	}

	userUUID := uuid.MustParse(userID)
	if err := ensureVersionIsOwnedRegisteredDraft(userUUID, version); err != nil {
		return nil, nil, uuid.Nil, err
	}

	return canvas, version, userUUID, nil
}

// ensureStagedReadAllowed restricts effective staged reads to the draft owner.
// Staging rows can outlive a draft's edit session; without this check any org
// reader could pass ?stage=true and read someone else's uncommitted work.
func ensureStagedReadAllowed(ctx context.Context, version *models.CanvasVersion) error {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	return ensureVersionIsOwnedRegisteredDraft(uuid.MustParse(userID), version)
}

// buildStagingSummary reports the uncommitted spec edits held in workflow_staged_files
// for a draft version so the UI can drive its orange/blue indicators.
func buildStagingSummary(versionID uuid.UUID, rows []models.WorkflowStaging) *pb.StagingSummary {
	state := &pb.StagingSummary{}
	if len(rows) == 0 {
		return state
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Path)
	}

	base := versionID.String()
	state.HasStaging = true
	state.StagedPaths = paths
	state.BaseVersionId = &base
	return state
}

// effectiveSpecYAML returns the YAML the UI should edit for a draft path:
// staged content when present, the materialized version row otherwise, and an
// empty string when the path is staged as deleted.
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

// StageRepositorySpecFileOperations stores repository file edits in
// workflow_staged_files verbatim, leaving workflow_versions untouched until commit.
// Both spec files (canvas.yaml/console.yaml, committed into the version row) and
// arbitrary repository files (committed to git) are accepted; the path kind is
// resolved at commit time.
func StageRepositorySpecFileOperations(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.StagingSummary, error) {
	canvas, version, userUUID, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	organizationUUID := canvas.OrganizationID

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
			if err := models.MarkWorkflowStagingPathDeleted(version.ID, organizationUUID, normalized, "", &userUUID); err != nil {
				return nil, grpcerrors.Internal(err, "failed to stage deletion")
			}
			continue
		}

		if _, err := models.UpsertWorkflowStagingPath(
			version.ID,
			organizationUUID,
			normalized,
			string(operation.GetContent()),
			"",
			&userUUID,
		); err != nil {
			return nil, grpcerrors.Internal(err, "failed to stage")
		}
	}

	rows, err := models.ListWorkflowStaging(version.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	publishStagingUpdated(canvas.ID, version.ID)

	return buildStagingSummary(version.ID, rows), nil
}

// stagingSummaryForVersion returns the StagingSummary for a version, used by reads
// to drive draft indicators without a dedicated list endpoint.
func stagingSummaryForVersion(versionID uuid.UUID) (*pb.StagingSummary, []models.WorkflowStaging, error) {
	rows, err := models.ListWorkflowStaging(versionID)
	if err != nil {
		return nil, nil, grpcerrors.Internal(err, "failed to load staging")
	}
	return buildStagingSummary(versionID, rows), rows, nil
}

// ReadStagedRepositoryFile returns the staged content for an arbitrary (non-spec)
// repository file on a draft version. found=false means there is no staging row
// for the path, so the caller should fall back to the committed git content.
// deleted=true means the file is staged for deletion.
func ReadStagedRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	path string,
) (content string, found bool, deleted bool, err error) {
	if strings.TrimSpace(versionID) == "" {
		return "", false, false, nil
	}

	_, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return "", false, false, err
	}

	if version.State != models.CanvasVersionStateDraft {
		return "", false, false, nil
	}

	if err := ensureStagedReadAllowed(ctx, version); err != nil {
		return "", false, false, err
	}

	_, rows, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return "", false, false, err
	}

	normalized := normalizeRepositoryFilePath(path)
	for _, row := range rows {
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
