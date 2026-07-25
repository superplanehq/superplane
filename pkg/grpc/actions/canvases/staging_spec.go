package canvases

import (
	"context"
	"slices"

	"github.com/google/uuid"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CanvasStagingState struct {
	Summary *pb.StagingSummary
	Spec    *pb.Canvas_Spec
}

func BuildCanvasStagingState(
	ctx context.Context,
	db *gorm.DB,
	registry *registry.Registry,
	canvas *models.Canvas,
	userID uuid.UUID,
) (*CanvasStagingState, error) {
	rows, err := models.ListStagedFilesForUser(db, canvas.ID, userID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	effectiveVersion, err := buildEffectiveCanvasVersion(db, registry, canvas, canvas.OrganizationID.String(), rows)
	if err != nil {
		return nil, err
	}

	return &CanvasStagingState{
		Summary: buildStagingSummary(canvas, rows),
		Spec:    serializeCanvasSpecFromVersion(effectiveVersion),
	}, nil
}

func validateStagedSpecFileContent(registry *registry.Registry, organizationID, path string, content []byte) error {
	normalized := normalizeRepositoryFilePath(path)
	switch normalized {
	case CanvasYAMLRepositoryPath:
		canvasDoc, err := yaml.CanvasFromYAML(content)
		if err != nil {
			return grpcerrors.InvalidArgument(err, "invalid canvas yaml")
		}

		_, _, err = canvasDoc.Parse(registry, organizationID)
		if err != nil {
			return grpcerrors.InvalidArgument(err, "invalid canvas yaml")
		}
	case ConsoleYAMLRepositoryPath:
		if _, err := yaml.ConsoleFromYML(content); err != nil {
			return grpcerrors.InvalidArgument(err, "invalid console yaml")
		}
	}

	return nil
}

func buildEffectiveCanvasVersion(
	db *gorm.DB,
	registry *registry.Registry,
	canvas *models.Canvas,
	organizationID string,
	stagedRows []models.WorkflowStagedFile,
) (*models.CanvasVersion, error) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load live version")
	}

	baseVersion := liveVersion
	if len(stagedRows) > 0 {
		baseVersion, err = models.FindCanvasVersionInTransaction(db, canvas.ID, stagedRows[0].BaseVersionID)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to load staging base version")
		}
	}

	effective := cloneCanvasVersionForStaging(baseVersion)
	stagedByPath := stagedSpecFilesByPath(stagedRows)

	if content, ok := stagedByPath[CanvasYAMLRepositoryPath]; ok {
		if err := applySpecFileContentToVersion(effective, liveVersion, registry, organizationID, CanvasYAMLRepositoryPath, content); err != nil {
			return nil, err
		}
	}

	if content, ok := stagedByPath[ConsoleYAMLRepositoryPath]; ok {
		if err := applySpecFileContentToVersion(effective, liveVersion, registry, organizationID, ConsoleYAMLRepositoryPath, content); err != nil {
			return nil, err
		}
	}

	return effective, nil
}

func stagedSpecFilesByPath(rows []models.WorkflowStagedFile) map[string][]byte {
	out := make(map[string][]byte)
	for _, row := range rows {
		if row.Deleted {
			continue
		}

		normalized := normalizeRepositoryFilePath(row.Path)
		if !IsRepositorySpecFilePath(normalized) {
			continue
		}

		out[normalized] = []byte(row.Content)
	}

	return out
}

func cloneCanvasVersionForStaging(base *models.CanvasVersion) *models.CanvasVersion {
	panels := base.ConsolePanels.Data()
	if panels == nil {
		panels = []models.ConsolePanel{}
	}

	layout := base.ConsoleLayout.Data()
	if layout == nil {
		layout = []models.ConsoleLayoutItem{}
	}

	return &models.CanvasVersion{
		ID:            base.ID,
		WorkflowID:    base.WorkflowID,
		Nodes:         datatypes.NewJSONSlice(slices.Clone(base.Nodes)),
		Edges:         datatypes.NewJSONSlice(slices.Clone(base.Edges)),
		ConsolePanels: datatypes.NewJSONType(slices.Clone(panels)),
		ConsoleLayout: datatypes.NewJSONType(slices.Clone(layout)),
	}
}

func applySpecFileContentToVersion(
	version *models.CanvasVersion,
	liveVersion *models.CanvasVersion,
	registry *registry.Registry,
	organizationID string,
	path string,
	content []byte,
) error {
	switch normalizeRepositoryFilePath(path) {
	case CanvasYAMLRepositoryPath:
		canvasDoc, err := yaml.CanvasFromYAML(content)
		if err != nil {
			return grpcerrors.Internal(err, "failed to parse staged canvas yaml")
		}

		nodes, edges, err := canvasDoc.Parse(registry, organizationID)
		if err != nil {
			return grpcerrors.Internal(err, "failed to parse staged canvas yaml")
		}

		version.Nodes = datatypes.NewJSONSlice(slices.Clone(injectMetadataIntoNodes(liveVersion.Nodes, nodes)))
		version.Edges = datatypes.NewJSONSlice(slices.Clone(edges))
	case ConsoleYAMLRepositoryPath:
		console, err := yaml.ConsoleFromYML(content)
		if err != nil {
			return grpcerrors.Internal(err, "failed to parse staged console yaml")
		}

		version.ConsolePanels = datatypes.NewJSONType(slices.Clone(console.Panels()))
		version.ConsoleLayout = datatypes.NewJSONType(slices.Clone(console.Layout()))
	default:
		return grpcerrors.InvalidArgument(nil, "unsupported repository spec file")
	}

	return nil
}
