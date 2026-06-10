package canvases

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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

func AppendRepositorySpecFilePaths(paths []string) []string {
	merged := make([]string, 0, len(paths)+2)
	seen := make(map[string]struct{}, len(paths)+2)

	for _, specPath := range []string{CanvasYAMLRepositoryPath, ConsoleYAMLRepositoryPath} {
		merged = append(merged, specPath)
		seen[specPath] = struct{}{}
	}

	for _, path := range paths {
		normalized := normalizeRepositoryFilePath(path)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		merged = append(merged, normalized)
	}

	sort.Strings(merged)
	return merged
}

func ReadRepositorySpecFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	path string,
) (string, error) {
	return readRepositorySpecFile(ctx, organizationID, canvasID, versionID, path, false)
}

// ReadRepositorySpecFileStaged returns the effective draft content for a spec
// path: staged content when present, the materialized version row otherwise.
func ReadRepositorySpecFileStaged(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	path string,
) (string, error) {
	return readRepositorySpecFile(ctx, organizationID, canvasID, versionID, path, true)
}

func readRepositorySpecFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	path string,
	stage bool,
) (string, error) {
	canvas, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return "", err
	}

	normalized := normalizeRepositoryFilePath(path)
	if normalized != CanvasYAMLRepositoryPath && normalized != ConsoleYAMLRepositoryPath {
		return "", status.Errorf(codes.InvalidArgument, "unsupported repository spec file %q", path)
	}

	if stage {
		_, rows, stagingErr := stagingStateForVersion(version.ID)
		if stagingErr != nil {
			return "", stagingErr
		}
		return effectiveSpecYAML(canvas, version, organizationID, rows, normalized)
	}

	switch normalized {
	case CanvasYAMLRepositoryPath:
		return canvasYAMLFromVersion(canvas, version, organizationID)
	default:
		return consoleYAMLFromVersion(version)
	}
}

func loadRepositorySpecVersionForRead(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
) (*models.Canvas, *models.CanvasVersion, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, nil, status.Error(codes.Internal, "failed to load canvas")
	}

	var version *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		resolvedVersionID, resolveErr := resolveConsoleVersionID(tx, canvas, strings.TrimSpace(versionID))
		if resolveErr != nil {
			return resolveErr
		}

		v, loadErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, resolvedVersionID)
		if loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return loadErr
		}

		if accessErr := ensureConsoleVersionReadable(ctx, tx, canvas, v); accessErr != nil {
			return accessErr
		}

		version = v
		return nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, nil, err
		}
		return nil, nil, status.Error(codes.Internal, "failed to load version")
	}

	return canvas, version, nil
}

// ApplyRepositorySpecFileOperations parses staged canvas.yaml/console.yaml
// content into the draft version row. It is the validated commit path shared by
// CommitCanvasStaging; positions come from the staged YAML, so no layout is
// applied here (auto-layout is its own RPC).
func ApplyRepositorySpecFileOperations(
	ctx context.Context,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
	operations []*pb.CanvasRepositoryFileOperation,
) error {
	if strings.TrimSpace(versionID) == "" {
		return status.Error(codes.InvalidArgument, "version_id is required for canvas.yaml and console.yaml updates")
	}

	for _, operation := range operations {
		if operation == nil {
			continue
		}
		if operation.GetDelete() {
			return status.Errorf(codes.InvalidArgument, "%q cannot be deleted", operation.GetPath())
		}

		normalized := normalizeRepositoryFilePath(operation.GetPath())
		content := string(operation.GetContent())

		switch normalized {
		case CanvasYAMLRepositoryPath:
			pbCanvas, err := canvasFromYAMLText(content)
			if err != nil {
				return err
			}

			_, err = UpdateCanvasVersionWithUsage(
				ctx,
				usageService,
				encryptor,
				registry,
				organizationID,
				canvasID,
				versionID,
				pbCanvas,
				nil,
				webhookBaseURL,
				authService,
			)
			if err != nil {
				return err
			}
		case ConsoleYAMLRepositoryPath:
			panels, layout, err := consolePanelsLayoutFromYAMLText(content)
			if err != nil {
				return err
			}

			_, err = UpdateConsole(ctx, organizationID, canvasID, versionID, panels, layout)
			if err != nil {
				return err
			}
		default:
			return status.Errorf(codes.InvalidArgument, "unsupported repository spec file %q", operation.GetPath())
		}
	}

	return nil
}
