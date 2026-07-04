package canvases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func effectiveSpecYAML(
	canvas *models.Canvas,
	version *models.CanvasVersion,
	organizationID string,
	rows []models.WorkflowStagedFile,
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

func ReadStagedRepositoryFile(
	ctx context.Context,
	db *gorm.DB,
	organizationID string,
	canvasID string,
	path string,
) (content string, found bool, deleted bool, err error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return "", false, false, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	stagedFiles, err := models.ListStagedFilesForUser(db, uuid.MustParse(canvasID), uuid.MustParse(userID))
	if err != nil {
		return "", false, false, err
	}

	normalized := normalizeRepositoryFilePath(path)
	for _, row := range stagedFiles {
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
	db *gorm.DB,
	organizationID string,
	canvasID string,
	version *models.CanvasVersion,
	path string,
) (string, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return "", grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvas, err := models.FindCanvasInTransaction(db, uuid.MustParse(organizationID), uuid.MustParse(canvasID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", grpcerrors.NotFound(err, "canvas not found")
		}
		return "", grpcerrors.Internal(err, "failed to load canvas")
	}

	stagedFiles, err := models.ListStagedFilesForUser(db, uuid.MustParse(canvasID), uuid.MustParse(userID))
	if err != nil {
		return "", err
	}

	return effectiveSpecYAML(canvas, version, organizationID, stagedFiles, path)
}
