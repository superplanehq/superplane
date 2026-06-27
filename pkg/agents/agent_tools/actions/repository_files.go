package actions

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	listFilesActionName   = "list_files"
	readFileActionName    = "read_file"
	writeFileActionName   = "write_file"
	deleteFileActionName  = "delete_file"
	commitFilesActionName = "commit_files"
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

	versionID, err := requestedReadableFileVersionID(session, input)
	if err != nil {
		return fileReadResult{}, err
	}

	result := fileReadResult{
		Action:   readFileActionName,
		CanvasID: session.CanvasID,
		Files:    make([]fileReadEntry, 0, len(paths)),
	}

	for _, path := range paths {
		entry, readErr := a.readPath(ctx, session, versionID, path)
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

func (a readFileAction) readPath(ctx context.Context, session agents.AgentSessionContext, versionID, path string) (fileReadEntry, error) {
	if canvasRepository.IsRepositorySpecFilePath(path) {
		source := "live"
		readSpecFile := canvasRepository.ReadRepositorySpecFile
		if versionID != "" {
			source = "draft"
			readSpecFile = canvasRepository.ReadRepositorySpecFileStaged
		}

		content, err := readSpecFile(ctx, session.OrganizationID, session.CanvasID, versionID, path)
		if err != nil {
			return fileReadEntry{}, err
		}

		return fileReadEntry{Path: path, Content: content, Source: source, VersionID: versionID}, nil
	}

	if versionID != "" {
		content, found, deleted, err := canvasRepository.ReadStagedRepositoryFile(
			ctx,
			session.OrganizationID,
			session.CanvasID,
			versionID,
			path,
		)
		if err != nil {
			return fileReadEntry{}, err
		}
		if deleted {
			return fileReadEntry{}, status.Errorf(codes.NotFound, "file %q is staged for deletion", path)
		}
		if found {
			return fileReadEntry{Path: path, Content: content, Source: "draft", VersionID: versionID}, nil
		}
	}

	content, err := a.readCommittedGitFile(ctx, session, path)
	if err != nil {
		return fileReadEntry{}, err
	}
	return fileReadEntry{Path: path, Content: content, Source: "live"}, nil
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

	draft, err := resolveFileDraftVersion(session, input)
	if err != nil {
		return fileStageResult{}, err
	}

	state, err := canvasRepository.StageRepositorySpecFileOperations(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		[]*pb.CanvasRepositoryFileOperation{{Path: path, Content: []byte(input.Content)}},
	)
	if err != nil {
		return fileStageResult{}, err
	}

	return fileStageResult{
		Action:         writeFileActionName,
		CanvasID:       session.CanvasID,
		VersionID:      draft.ID.String(),
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

	draft, err := resolveFileDraftVersion(session, input)
	if err != nil {
		return fileStageResult{}, err
	}

	state, err := canvasRepository.StageRepositorySpecFileOperations(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		[]*pb.CanvasRepositoryFileOperation{{Path: path, Delete: true}},
	)
	if err != nil {
		return fileStageResult{}, err
	}

	return fileStageResult{
		Action:         deleteFileActionName,
		CanvasID:       session.CanvasID,
		VersionID:      draft.ID.String(),
		Path:           path,
		Deleted:        true,
		StagingSummary: serializeStagingSummary(state),
	}, nil
}

type commitFilesAction struct {
	deps Dependencies
}

func newCommitFilesAction(deps Dependencies) commitFilesAction {
	return commitFilesAction{deps: deps}
}

func (commitFilesAction) Name() string {
	return commitFilesActionName
}

func (a commitFilesAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	if a.deps.GitProvider == nil {
		return fileCommitResult{}, fmt.Errorf("git provider is not configured")
	}

	draft, err := resolveFileDraftVersion(session, input)
	if err != nil {
		return fileCommitResult{}, err
	}

	response, err := canvasRepository.CommitCanvasStaging(
		ctx,
		a.deps.GitProvider,
		a.deps.UsageService,
		a.deps.Encryptor,
		a.deps.Registry,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		a.deps.WebhookBaseURL,
		a.deps.AuthService,
		strings.TrimSpace(input.Message),
	)
	if err != nil {
		return fileCommitResult{}, err
	}

	return fileCommitResult{
		Action:    commitFilesActionName,
		CanvasID:  session.CanvasID,
		VersionID: draft.ID.String(),
		Draft: draftResult{
			VersionID:   draft.ID.String(),
			DisplayName: draft.DisplayName,
			BranchName:  draft.GitBranch,
		},
		StagingSummary: serializeStagingSummary(response.GetStagingSummary()),
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
	if canvasRepository.IsRepositorySpecFilePath(path) {
		return "", fmt.Errorf("use patch_draft or update_draft for %s", path)
	}
	return path, nil
}

func requestedReadableFileVersionID(session agents.AgentSessionContext, input Input) (string, error) {
	if input.UseDraft != nil && !*input.UseDraft {
		return "", nil
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return "", err
	}
	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return "", err
	}

	draft, err := resolveReadableDraftVersion(canvasID, userID, input)
	if err != nil {
		return "", err
	}
	if draft == nil {
		return "", nil
	}
	return draft.ID.String(), nil
}

func resolveFileDraftVersion(session agents.AgentSessionContext, input Input) (*models.CanvasVersion, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid session canvas id: %w", err)
	}
	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid session user id: %w", err)
	}
	return resolveTargetDraftVersion(canvasID, userID, input)
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
