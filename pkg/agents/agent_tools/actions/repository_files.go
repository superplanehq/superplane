package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/services/files"
)

const (
	listFilesActionName = "list_files"
	readFileActionName  = "read_file"
)

type listFilesAction struct{}

func newListFilesAction() listFilesAction {
	return listFilesAction{}
}

func (listFilesAction) Name() string {
	return listFilesActionName
}

func (listFilesAction) Execute(_ context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	orgID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return fileListResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return fileListResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	if _, err := models.FindCanvas(orgID, canvasID); err != nil {
		return fileListResult{}, fmt.Errorf("find canvas: %w", err)
	}

	query := strings.ToLower(strings.TrimSpace(input.Query))
	listed := make([]string, 0, 2)
	for _, path := range []string{files.CanvasYAMLPath, files.ConsoleYAMLPath} {
		if query != "" && !strings.Contains(strings.ToLower(path), query) {
			continue
		}
		listed = append(listed, path)
	}

	return fileListResult{
		Action:   listFilesActionName,
		CanvasID: session.CanvasID,
		Files:    listed,
	}, nil
}

type readFileAction struct{}

func newReadFileAction() readFileAction {
	return readFileAction{}
}

func (readFileAction) Name() string {
	return readFileActionName
}

func (a readFileAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	orgID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return fileReadResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return fileReadResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	paths, err := requestedFilePaths(input)
	if err != nil {
		return fileReadResult{}, err
	}

	versionID, err := requestedReadableFileVersionID(session, input)
	if err != nil {
		return fileReadResult{}, err
	}

	result := fileReadResult{
		Action:   readFileActionName,
		CanvasID: session.CanvasID,
		Files:    make([]fileReadEntry, 0, len(paths)),
	}

	canvas, err := models.FindCanvas(orgID, canvasID)
	if err != nil {
		return fileReadEntry{}, fmt.Errorf("find canvas: %w", err)
	}

	for _, path := range paths {
		entry, readErr := a.readPath(ctx, canvas, versionID, path)
		if readErr != nil {
			result.Errors = append(result.Errors, fileReadError{Path: path, Error: readErr.Error()})
			continue
		}
		result.Files = append(result.Files, entry)
	}

	if len(result.Files) == 0 && len(result.Errors) > 0 {
		return fileReadResult{}, fmt.Errorf("read files: %s", result.Errors[0].Error)
	}

	return result, nil
}

func (a readFileAction) readPath(ctx context.Context, canvas *models.Canvas, versionID string, path string) (fileReadEntry, error) {
	parsedVersionID, err := uuid.Parse(versionID)
	if err != nil {
		return fileReadEntry{}, fmt.Errorf("invalid version id: %w", err)
	}

	version, err := models.FindCanvasVersion(canvas.ID, parsedVersionID)
	if err != nil {
		return fileReadEntry{}, fmt.Errorf("find canvas version: %w", err)
	}

	content, err := canvasRepository.ReadRepositorySpecFileStaged(ctx, canvas, version, path)
	if err != nil {
		return fileReadEntry{}, err
	}

	return fileReadEntry{Path: path, Content: content, Source: "staging", VersionID: parsedVersionID.String()}, nil
}

func requestedFilePaths(input Input) ([]string, error) {
	rawPaths := append([]string(nil), input.Paths...)
	if strings.TrimSpace(input.Path) != "" {
		rawPaths = append(rawPaths, input.Path)
	}
	if len(rawPaths) == 0 {
		return nil, fmt.Errorf("path or paths is required for read_file")
	}

	paths := make([]string, 0, len(rawPaths))
	seen := map[string]struct{}{}
	for _, rawPath := range rawPaths {
		path := files.NormalizePath(rawPath)
		if !files.IsSpecFilePath(path) {
			return nil, fmt.Errorf("invalid file path %q: only canvas.yaml and console.yaml are readable", rawPath)
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths, nil
}

func requestedReadableFileVersionID(session agents.AgentSessionContext, input Input) (string, error) {
	liveVersion, err := resolveFileLiveVersion(session, input)
	if err != nil {
		return "", err
	}
	return liveVersion.ID.String(), nil
}

func resolveFileLiveVersion(session agents.AgentSessionContext, input Input) (*models.CanvasVersion, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid session canvas id: %w", err)
	}
	return resolveLiveCanvasVersion(canvasID, input)
}
