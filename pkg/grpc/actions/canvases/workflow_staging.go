package canvases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func effectiveSpecYAML(
	canvas *models.Canvas,
	version *models.CanvasVersion,
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
		return yaml.VersionToCanvasYAML(canvas.Name, canvas.Description, version)
	case ConsoleYAMLRepositoryPath:
		return yaml.VersionToConsoleYML(canvas.Name, version)
	default:
		return "", grpcerrors.InvalidArgument(nil, fmt.Sprintf("unsupported repository spec file %q", path))
	}
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

	return effectiveSpecYAML(canvas, version, stagedFiles, path)
}
