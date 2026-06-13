package canvases

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
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
		if err := ensureStagedReadAllowed(ctx, version); err != nil {
			return "", err
		}

		_, rows, stagingErr := stagingSummaryForVersion(version.ID)
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

// ApplyRepositorySpecFileOperations commits canvas.yaml/console.yaml to the draft
// branch in git, materializes the resulting commit, and optionally clears staging.
func ApplyRepositorySpecFileOperations(
	ctx context.Context,
	gitProvider git.Provider,
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
) error {
	if strings.TrimSpace(versionID) == "" {
		return status.Error(codes.InvalidArgument, "version_id is required for canvas.yaml and console.yaml updates")
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	version, err := models.FindCanvasVersion(canvasUUID, versionUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "version not found")
		}
		return status.Errorf(codes.Internal, "failed to load version: %v", err)
	}

	processed := make([]*pb.CanvasRepositoryFileOperation, 0, len(operations))
	for _, operation := range operations {
		if operation == nil {
			continue
		}
		if operation.GetDelete() {
			return status.Errorf(codes.InvalidArgument, "%q cannot be deleted", operation.GetPath())
		}

		normalized := normalizeRepositoryFilePath(operation.GetPath())
		content := string(operation.GetContent())

		if normalized == CanvasYAMLRepositoryPath && autoLayout != nil {
			pbCanvas, parseErr := canvasFromYAMLText(content)
			if parseErr != nil {
				return parseErr
			}

			nodes := actions.ProtoToNodes(pbCanvas.GetSpec().GetNodes())
			edges := actions.ProtoToEdges(pbCanvas.GetSpec().GetEdges())
			laidOutNodes, laidOutEdges, layoutErr := layout.ApplyLayout(nodes, edges, autoLayout)
			if layoutErr != nil {
				return status.Errorf(codes.InvalidArgument, "failed to apply layout: %v", layoutErr)
			}

			positioned := &pb.CanvasVersion{
				Metadata: &pb.CanvasVersion_Metadata{
					Name:        pbCanvas.GetMetadata().GetName(),
					Description: pbCanvas.GetMetadata().GetDescription(),
				},
				Spec: &pb.Canvas_Spec{
					Nodes:            actions.NodesToProto(laidOutNodes),
					Edges:            actions.EdgesToProto(laidOutEdges),
					ChangeManagement: pbCanvas.GetSpec().GetChangeManagement(),
				},
			}

			positionedYAML, yamlErr := canvasyaml.CanvasResourceYAML(positioned, canvasID)
			if yamlErr != nil {
				return status.Errorf(codes.Internal, "failed to serialize canvas: %v", yamlErr)
			}
			content = positionedYAML
		}

		processed = append(processed, &pb.CanvasRepositoryFileOperation{
			Path:    normalized,
			Content: []byte(content),
		})
	}

	if len(processed) == 0 {
		return status.Error(codes.InvalidArgument, "at least one file operation is required")
	}

	_, err = CommitCanvasRepositoryFiles(
		ctx,
		gitProvider,
		usageService,
		encryptor,
		registry,
		organizationID,
		canvasID,
		versionID,
		version.CommitSHA,
		"Update repository spec files",
		processed,
		autoLayout,
		webhookBaseURL,
		authService,
	)
	if err != nil {
		return err
	}

	if discardStaging {
		if discardErr := models.DiscardWorkflowStaging(versionUUID, nil); discardErr != nil {
			return status.Errorf(codes.Internal, "failed to discard staging: %v", discardErr)
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

func resolveCommitCanvasAutoLayout(hasAutoLayout bool, autoLayout *pb.CanvasAutoLayout) *pb.CanvasAutoLayout {
	if !hasAutoLayout {
		return nil
	}
	if autoLayout == nil {
		return nil
	}
	if autoLayout.Algorithm == pb.CanvasAutoLayout_ALGORITHM_UNSPECIFIED &&
		autoLayout.Scope == pb.CanvasAutoLayout_SCOPE_UNSPECIFIED &&
		len(autoLayout.NodeIds) == 0 {
		return nil
	}
	return autoLayout
}

func splitRepositoryFileOperations(operations []*pb.CanvasRepositoryFileOperation) (specOps []*pb.CanvasRepositoryFileOperation, gitOps []*pb.CanvasRepositoryFileOperation) {
	for _, operation := range operations {
		if operation == nil {
			continue
		}
		if IsRepositorySpecFilePath(operation.GetPath()) {
			specOps = append(specOps, operation)
			continue
		}
		gitOps = append(gitOps, operation)
	}
	return specOps, gitOps
}
