package actions

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/services/files"
)

const (
	listFilesActionName  = "list_files"
	readFileActionName   = "read_file"
	writeFileActionName  = "write_file"
	deleteFileActionName = "delete_file"
)

type listFilesAction struct {
	gitProvider gitprovider.Provider
}

func newListFilesAction(deps Dependencies) listFilesAction {
	return listFilesAction{gitProvider: deps.GitProvider}
}

func (listFilesAction) Name() string {
	return listFilesActionName
}

func (a listFilesAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	if a.gitProvider == nil {
		return fileListResult{}, fmt.Errorf("git provider is not configured")
	}

	response, err := canvasRepository.ListCanvasRepositoryFiles(ctx, a.gitProvider, session.OrganizationID, session.CanvasID)
	if err != nil {
		return fileListResult{}, err
	}

	query := strings.ToLower(strings.TrimSpace(input.Query))
	files := make([]string, 0, len(response.GetFiles()))
	for _, file := range response.GetFiles() {
		path := file.GetPath()
		if query != "" && !strings.Contains(strings.ToLower(path), query) {
			continue
		}
		files = append(files, path)
	}
	sort.Strings(files)

	return fileListResult{
		Action:       listFilesActionName,
		CanvasID:     session.CanvasID,
		Files:        files,
		ContextFiles: contextFilePaths(files),
	}, nil
}

type readFileAction struct {
	gitProvider gitprovider.Provider
}

func newReadFileAction(deps Dependencies) readFileAction {
	return readFileAction{gitProvider: deps.GitProvider}
}

func (readFileAction) Name() string {
	return readFileActionName
}

func (a readFileAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	if a.gitProvider == nil {
		return fileReadResult{}, fmt.Errorf("git provider is not configured")
	}

	paths, err := requestedFilePaths(input)
	if err != nil {
		return fileReadResult{}, err
	}

	result := fileReadResult{
		Action:   readFileActionName,
		CanvasID: session.CanvasID,
		Files:    make([]fileReadEntry, 0, len(paths)),
	}

	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return fileReadResult{}, fmt.Errorf("invalid session user id: %w", err)
	}

	for _, path := range paths {
		entry, readErr := a.readPath(ctx, session, userID, path)
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

func (a readFileAction) readPath(ctx context.Context, session agents.AgentSessionContext, userID uuid.UUID, path string) (fileReadEntry, error) {
	db := database.DB(ctx)

	canvas, err := models.FindCanvasInTransaction(db, uuid.MustParse(session.OrganizationID), uuid.MustParse(session.CanvasID))
	if err != nil {
		return fileReadEntry{}, err
	}

	fileReader := files.NewAppFileReader(db, canvas, userID)
	r, err := fileReader.ReadFromStaging(ctx, path)
	if err == nil {
		content, err := io.ReadAll(r)
		if err != nil {
			return fileReadEntry{}, err
		}

		return fileReadEntry{Path: path, Content: string(content), Source: "staging"}, nil
	}

	r, err = fileReader.ReadFromVersion(ctx, canvas.LiveVersionID.String(), path)
	if err != nil {
		return fileReadEntry{}, err
	}

	content, err := io.ReadAll(r)
	if err != nil {
		return fileReadEntry{}, err
	}

	return fileReadEntry{Path: path, Content: string(content), Source: "live", VersionID: canvas.LiveVersionID.String()}, nil
}

func (a readFileAction) readCommittedGitFile(ctx context.Context, session agents.AgentSessionContext, path string) (string, error) {
	orgID, canvasID, err := parseSessionIDs(session)
	if err != nil {
		return "", err
	}

	repository, err := models.FindRepository(orgID, canvasID)
	if err != nil {
		return "", fmt.Errorf("repository not found: %w", err)
	}

	reader, err := a.gitProvider.GetFile(ctx, repository.RepoID, path, "")
	if err != nil {
		return "", fmt.Errorf("read repository file %q: %w", path, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read repository file %q: %w", path, err)
	}
	return string(content), nil
}

type writeFileAction struct{}

func (writeFileAction) Name() string {
	return writeFileActionName
}

func (writeFileAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	path, err := requestedWritableFilePath(input.Path)
	if err != nil {
		return fileStageResult{}, err
	}

	organizationID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return fileStageResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return fileStageResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	db := database.DB(ctx)
	canvas, err := models.FindCanvasInTransaction(db, organizationID, canvasID)
	if err != nil {
		return fileStageResult{}, err
	}

	liveVersion, err := models.FindCanvasVersionInTransaction(db, canvasID, *canvas.LiveVersionID)
	if err != nil {
		return fileStageResult{}, fmt.Errorf("load live version: %w", err)
	}

	state, err := canvasRepository.PutCanvasStaging(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		[]*pb.CanvasRepositoryFileOperation{{Path: path, Content: []byte(input.Content)}},
	)
	if err != nil {
		return fileStageResult{}, err
	}

	return fileStageResult{
		Action:         writeFileActionName,
		CanvasID:       session.CanvasID,
		VersionID:      liveVersion.ID.String(),
		Path:           path,
		StagingSummary: serializeStagingSummary(state),
	}, nil
}

type deleteFileAction struct{}

func (deleteFileAction) Name() string {
	return deleteFileActionName
}

func (deleteFileAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	path, err := requestedWritableFilePath(input.Path)
	if err != nil {
		return fileStageResult{}, err
	}

	organizationID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return fileStageResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return fileStageResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	db := database.DB(ctx)
	canvas, err := models.FindCanvasInTransaction(db, organizationID, canvasID)
	if err != nil {
		return fileStageResult{}, err
	}

	liveVersion, err := models.FindCanvasVersionInTransaction(db, canvasID, *canvas.LiveVersionID)
	if err != nil {
		return fileStageResult{}, fmt.Errorf("load live version: %w", err)
	}

	state, err := canvasRepository.PutCanvasStaging(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		[]*pb.CanvasRepositoryFileOperation{{Path: path, Delete: true}},
	)
	if err != nil {
		return fileStageResult{}, err
	}

	return fileStageResult{
		Action:         deleteFileActionName,
		CanvasID:       session.CanvasID,
		VersionID:      liveVersion.ID.String(),
		Path:           path,
		Deleted:        true,
		StagingSummary: serializeStagingSummary(state),
	}, nil
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
		path, err := gitprovider.ValidateUserPath(rawPath)
		if err != nil {
			return nil, fmt.Errorf("invalid file path %q: %w", rawPath, err)
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths, nil
}

func requestedWritableFilePath(rawPath string) (string, error) {
	path, err := gitprovider.ValidateUserPath(rawPath)
	if err != nil {
		return "", fmt.Errorf("invalid file path %q: %w", rawPath, err)
	}
	if files.IsSpecFilePath(path) {
		return "", fmt.Errorf("use patch_staging for %s", path)
	}
	return path, nil
}

func parseSessionIDs(session agents.AgentSessionContext) (uuid.UUID, uuid.UUID, error) {
	orgID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid session organization id: %w", err)
	}
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid session canvas id: %w", err)
	}
	return orgID, canvasID, nil
}

func serializeStagingSummary(summary *pb.StagingSummary) stagingSummary {
	if summary == nil {
		return stagingSummary{}
	}
	return stagingSummary{
		HasStaging:  summary.GetHasStaging(),
		StagedPaths: append([]string(nil), summary.GetStagedPaths()...),
	}
}

func contextFilePaths(paths []string) []string {
	matches := []string{}
	for _, path := range paths {
		if isContextFilePath(path) {
			matches = append(matches, path)
		}
	}
	return matches
}

func isContextFilePath(path string) bool {
	base := strings.ToLower(path)
	if index := strings.LastIndex(base, "/"); index >= 0 {
		base = base[index+1:]
	}
	switch base {
	case "agents.md", "agent.md", "claude.md", "readme.md":
		return true
	default:
		return strings.HasSuffix(base, ".agents.md")
	}
}
