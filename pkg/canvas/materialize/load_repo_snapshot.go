package materialize

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
)

type repoSnapshot struct {
	Name          string
	Description   string
	Nodes         []models.Node
	Edges         []models.Edge
	ConsolePanels []models.ConsolePanel
	ConsoleLayout []models.ConsoleLayoutItem
}

func loadRepoSnapshot(
	ctx context.Context,
	gitProvider git.Provider,
	registry *registry.Registry,
	orgID uuid.UUID,
	repoID string,
	sha string,
) (*repoSnapshot, error) {
	canvasYAML, err := readGitFile(ctx, gitProvider, repoID, models.CanvasFileName, sha)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", models.CanvasFileName, err)
	}

	pbCanvas, err := canvasyaml.ParseCanvasResource(canvasYAML)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", models.CanvasFileName, err)
	}

	nodes, edges, err := snapshotNodesAndEdges(pbCanvas)
	if err != nil {
		return nil, fmt.Errorf("validate canvas spec: %w", err)
	}
	_ = registry
	_ = orgID

	snapshot := &repoSnapshot{
		Name:        pbCanvas.GetMetadata().GetName(),
		Description: pbCanvas.GetMetadata().GetDescription(),
		Nodes:       nodes,
		Edges:       edges,
	}

	consoleYAML, err := readGitFile(ctx, gitProvider, repoID, models.ConsoleFileName, sha)
	if err != nil {
		if errors.Is(err, errGitFileNotFound) {
			return snapshot, nil
		}
		return nil, fmt.Errorf("read %s: %w", models.ConsoleFileName, err)
	}

	console, err := models.ConsoleFromYML(consoleYAML)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", models.ConsoleFileName, err)
	}

	snapshot.ConsolePanels = console.Spec.Panels
	snapshot.ConsoleLayout = console.Spec.Layout
	return snapshot, nil
}

var errGitFileNotFound = errors.New("git file not found")

func readGitFile(ctx context.Context, gitProvider git.Provider, repoID, path, ref string) ([]byte, error) {
	reader, err := gitProvider.GetFile(ctx, repoID, path, ref)
	if err != nil {
		if errors.Is(err, git.ErrInvalidRef) || errors.Is(err, git.ErrInvalidPath) {
			return nil, fmt.Errorf("%w: %s", errGitFileNotFound, path)
		}
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: %s", errGitFileNotFound, path)
	}

	return data, nil
}

func snapshotNodesAndEdges(canvas *pb.Canvas) ([]models.Node, []models.Edge, error) {
	if canvas.GetSpec() == nil {
		return []models.Node{}, []models.Edge{}, nil
	}

	return actions.ProtoToNodes(canvas.GetSpec().GetNodes()), actions.ProtoToEdges(canvas.GetSpec().GetEdges()), nil
}
