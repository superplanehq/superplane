package canvases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func publishStagingUpdated(canvasID, versionID uuid.UUID) {
	if err := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), versionID.String()).PublishStagingUpdated(); err != nil {
		log.Errorf("failed to publish canvas staging updated RabbitMQ message: %v", err)
	}
}

func buildStagingSummary(branchID uuid.UUID, rows []models.WorkflowStaging) *pb.StagingSummary {
	state := &pb.StagingSummary{}
	if len(rows) == 0 {
		return state
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Path)
	}

	base := branchID.String()
	state.HasStaging = true
	state.StagedPaths = paths
	state.BaseVersionId = &base
	return state
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

func StageRepositorySpecFileOperations(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	branchName string,
	operations []*pb.CanvasRepositoryFileOperation,
) (*pb.StagingSummary, error) {
	canvas, branch, headVersion, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, branchName, versionID)
	if err != nil {
		return nil, err
	}

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
			if err := models.MarkWorkflowStagingPathDeleted(branch.ID, userUUID, normalized, &userUUID); err != nil {
				return nil, grpcerrors.Internal(err, "failed to stage deletion")
			}
			continue
		}

		if _, err := models.UpsertWorkflowStagingPath(
			branch.ID,
			userUUID,
			normalized,
			string(operation.GetContent()),
			&userUUID,
		); err != nil {
			return nil, grpcerrors.Internal(err, "failed to stage")
		}
	}

	rows, err := models.ListWorkflowStaging(branch.ID, userUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	publishStagingUpdated(canvas.ID, headVersion.ID)

	return buildStagingSummary(branch.ID, rows), nil
}

func stagingSummaryForBranch(branchID, userID uuid.UUID) (*pb.StagingSummary, []models.WorkflowStaging, error) {
	rows, err := models.ListWorkflowStaging(branchID, userID)
	if err != nil {
		return nil, nil, grpcerrors.Internal(err, "failed to load staging")
	}
	return buildStagingSummary(branchID, rows), rows, nil
}

func stagingSummaryForVersion(versionID uuid.UUID) (*pb.StagingSummary, []models.WorkflowStaging, error) {
	_ = versionID
	return &pb.StagingSummary{}, nil, nil
}

func ReadStagedRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	branchName string,
	path string,
) (content string, found bool, deleted bool, err error) {
	if strings.TrimSpace(versionID) == "" && strings.TrimSpace(branchName) == "" {
		return "", false, false, nil
	}

	_, branch, _, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, branchName, versionID)
	if err != nil {
		return "", false, false, err
	}

	_, rows, err := stagingSummaryForBranch(branch.ID, userUUID)
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

func ensureStagedReadAllowed(ctx context.Context, branchID uuid.UUID) error {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	_ = branchID
	return nil
}

// Legacy adapter used by older call sites during POC transition.
func loadOwnedDraftVersion(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*models.Canvas, *models.CanvasVersion, uuid.UUID, error) {
	canvas, _, headVersion, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, "", versionID)
	if err != nil {
		return nil, nil, uuid.Nil, err
	}
	return canvas, headVersion, userUUID, nil
}
