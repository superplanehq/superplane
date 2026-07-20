package canvases

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	CanvasYAMLRepositoryPath  = "canvas.yaml"
	ConsoleYAMLRepositoryPath = "console.yaml"
)

func IsRepositorySpecFilePath(path string) bool {
	normalized := normalizeRepositoryFilePath(path)
	return normalized == CanvasYAMLRepositoryPath || normalized == ConsoleYAMLRepositoryPath
}

func normalizeRepositoryFilePath(path string) string {
	return strings.TrimLeft(strings.TrimSpace(strings.ReplaceAll(path, "\\", "/")), "/")
}

func ReadRepositorySpecFile(ctx context.Context, canvas *models.Canvas, version *models.CanvasVersion, path string) (string, error) {
	return readRepositorySpecFile(ctx, canvas, version, path, false)
}

// ReadRepositorySpecFileStaged returns the effective draft content for a spec
// path: staged content when present, the materialized version row otherwise.
func ReadRepositorySpecFileStaged(ctx context.Context, canvas *models.Canvas, version *models.CanvasVersion, path string) (string, error) {
	return readRepositorySpecFile(ctx, canvas, version, path, true)
}

func readRepositorySpecFile(ctx context.Context, canvas *models.Canvas, version *models.CanvasVersion, path string, stage bool) (string, error) {
	db := database.DB(ctx)
	normalized := normalizeRepositoryFilePath(path)
	if normalized != CanvasYAMLRepositoryPath && normalized != ConsoleYAMLRepositoryPath {
		return "", grpcerrors.InvalidArgument(nil, fmt.Sprintf("unsupported repository spec file %q", path))
	}

	if stage {
		return ReadStagedRepositorySpecFile(ctx, db, canvas.OrganizationID.String(), canvas.ID.String(), version, normalized)
	}

	switch normalized {
	case CanvasYAMLRepositoryPath:
		return yaml.VersionToCanvasYAML(
			canvas.Name,
			canvas.Description,
			version,
		)
	default:
		return yaml.VersionToConsoleYML(canvas.Name, version)
	}
}
