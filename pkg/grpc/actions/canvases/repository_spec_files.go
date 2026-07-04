package canvases

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
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
	db := database.DB(ctx)
	canvas, version, err := loadRepositorySpecVersionForRead(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return "", err
	}

	normalized := normalizeRepositoryFilePath(path)
	if normalized != CanvasYAMLRepositoryPath && normalized != ConsoleYAMLRepositoryPath {
		return "", grpcerrors.InvalidArgument(nil, fmt.Sprintf("unsupported repository spec file %q", path))
	}

	if stage {
		return ReadStagedRepositorySpecFile(ctx, db, organizationID, canvasID, version, normalized)
	}

	switch normalized {
	case CanvasYAMLRepositoryPath:
		return canvasYAMLFromVersion(canvas, version, organizationID)
	default:
		return consoleYAMLFromVersion(canvas, version)
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
		return nil, nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, nil, grpcerrors.InvalidArgument(nil, "invalid canvas_id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	var version *models.CanvasVersion
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		resolvedVersionID, resolveErr := resolveLiveCanvasVersionID(tx, canvas, versionID)
		if resolveErr != nil {
			return resolveErr
		}

		v, loadErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, resolvedVersionID)
		if loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				return grpcerrors.NotFound(loadErr, "version not found")
			}
			return loadErr
		}

		version = v
		return nil
	})
	if err != nil {
		if grpcerrors.Code(err) != codes.Unknown {
			return nil, nil, err
		}
		return nil, nil, grpcerrors.Internal(err, "failed to load version")
	}

	return canvas, version, nil
}

// ApplyRepositorySpecFileOperationsToCommitTarget parses canvas.yaml/console.yaml
// content into a new commit version row during staging commit.
func ApplyRepositorySpecFileOperationsToCommitTarget(
	ctx context.Context,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
	autoLayout *pb.CanvasAutoLayout,
	operations []*pb.CanvasRepositoryFileOperation,
) error {
	return applyRepositorySpecFileOperations(
		ctx,
		usageService,
		encryptor,
		registry,
		organizationID,
		canvasID,
		versionID,
		webhookBaseURL,
		authService,
		autoLayout,
		false,
		operations,
		true,
	)
}

func applyRepositorySpecFileOperations(
	ctx context.Context,
	usageService usage.Service,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	webhookBaseURL string,
	authService authorization.Authorization,
	autoLayout *pb.CanvasAutoLayout,
	discardStaging bool,
	operations []*pb.CanvasRepositoryFileOperation,
	commitTarget bool,
) error {
	if strings.TrimSpace(versionID) == "" {
		return grpcerrors.InvalidArgument(nil, "version_id is required for canvas.yaml and console.yaml updates")
	}

	for _, operation := range operations {
		if operation == nil {
			continue
		}
		if operation.GetDelete() {
			return grpcerrors.InvalidArgument(nil, fmt.Sprintf("%q cannot be deleted", operation.GetPath()))
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
				autoLayout,
				webhookBaseURL,
				authService,
				discardStaging,
				commitTarget,
			)
			if err != nil {
				return err
			}
		case ConsoleYAMLRepositoryPath:
			panels, layout, err := consolePanelsLayoutFromYAMLText(content)
			if err != nil {
				return err
			}

			_, err = UpdateConsole(ctx, organizationID, canvasID, versionID, panels, layout, discardStaging, commitTarget)
			if err != nil {
				return err
			}
		default:
			return grpcerrors.InvalidArgument(nil, fmt.Sprintf("unsupported repository spec file %q", operation.GetPath()))
		}
	}

	return nil
}

// ParseAndValidateCanvasYAML parses canvas.yaml text and runs the same registry
// validation as the commit path, returning materialized nodes/edges (carrying
// per-node error/warning messages) without persisting anything. Agent tools use
// it to validate staged edits before staging and to summarize staged content.
func ParseAndValidateCanvasYAML(registry *registry.Registry, organizationID, text string) ([]models.Node, []models.Edge, error) {
	pbCanvas, err := canvasFromYAMLText(text)
	if err != nil {
		return nil, nil, err
	}
	return ParseCanvas(registry, organizationID, pbCanvas)
}

// ValidateConsoleYAML parses and validates console.yaml text without persisting,
// mirroring the validation the commit path runs before writing the version row.
func ValidateConsoleYAML(text string) error {
	_, _, err := consolePanelsLayoutFromYAMLText(text)
	return err
}
