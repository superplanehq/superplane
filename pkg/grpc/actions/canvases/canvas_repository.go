package canvases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/git"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type CanvasRepositoryStorageOptions struct {
	ProviderName  string
	DefaultBranch string
	MaxFileBytes  int64
}

func GetCanvasRepository(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.GetCanvasRepositoryResponse, error) {
	repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	return &pb.GetCanvasRepositoryResponse{Repository: serializeCanvasRepository(ctx, repository, storage)}, nil
}

func ListCanvasRepositoryFiles(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.ListCanvasRepositoryFilesResponse, error) {
	repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	result, err := storage.ListFiles(ctx, canvasRepositoryRef(repository), git.ListFilesOptions{Ref: "main"})
	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	files := make([]*pb.CanvasRepositoryFile, 0, len(result.Paths))
	for _, path := range result.Paths {
		files = append(files, &pb.CanvasRepositoryFile{Path: path})
	}

	return &pb.ListCanvasRepositoryFilesResponse{
		Files: files,
	}, nil
}

type OpenedCanvasRepositoryFile struct {
	Path    string
	Content io.ReadCloser
}

func OpenCanvasRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	path string,
	ref string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*OpenedCanvasRepositoryFile, error) {
	repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	requestPath, err := url.PathUnescape(path)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file path")
	}

	normalizedPath, err := git.NormalizePath(requestPath)
	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	reader, err := storage.GetFile(ctx, canvasRepositoryRef(repository), git.GetFileOptions{
		Path: normalizedPath,
		Ref:  ref,
	})
	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	return &OpenedCanvasRepositoryFile{
		Path:    normalizedPath,
		Content: reader,
	}, nil
}

func CommitCanvasRepositoryFiles(
	ctx context.Context,
	organizationID string,
	canvasID string,
	req *pb.CommitCanvasRepositoryFilesRequest,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.CommitCanvasRepositoryFilesResponse, error) {
	repository, canvas, err := ensureWritableCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	author, err := canvasRepositoryCommitAuthor(ctx, canvas.OrganizationID.String())
	if err != nil {
		return nil, err
	}

	operations := make([]git.FileOperation, 0, len(req.GetOperations()))
	for _, operation := range req.GetOperations() {
		content := operation.GetContent()
		var reader io.Reader
		if !operation.GetDelete() {
			reader = bytes.NewReader(content)
		}

		operations = append(operations, git.FileOperation{
			Path:      operation.GetPath(),
			Content:   reader,
			SizeBytes: int64(len(content)),
			Delete:    operation.GetDelete(),
		})
	}

	result, err := storage.Commit(ctx, canvasRepositoryRef(repository), git.CommitOptions{
		Branch:          "main",
		BaseBranch:      "main",
		ExpectedHeadSHA: req.GetExpectedHeadSha(),
		Message:         req.GetMessage(),
		Author:          author,
		Operations:      operations,
	})

	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	return &pb.CommitCanvasRepositoryFilesResponse{
		CommitSha: result.CommitSHA,
	}, nil
}

func ensureCanvasRepository(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*models.CanvasRepository, error) {
	if storage == nil {
		return nil, status.Error(codes.FailedPrecondition, "canvas file storage is not configured")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	canvas, err := models.FindCanvas(organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	repository, err := models.FindCanvasRepository(canvas.ID)
	if err == nil {
		return repository, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.Internal, "failed to load canvas repository")
	}

	storageRepo, err := storage.CreateRepository(ctx, git.RepositorySpec{
		OrganizationID: organizationUUID,
		CanvasID:       canvas.ID,
		RepoID:         git.CanvasRepoID(organizationUUID, canvas.ID),
		DefaultBranch:  options.DefaultBranch,
	})

	if err != nil {
		return nil, gitStorageStatusError(err)
	}

	now := time.Now()
	repository = &models.CanvasRepository{
		CanvasID:       canvas.ID,
		OrganizationID: organizationUUID,
		Provider:       options.ProviderName,
		RepoID:         storageRepo.RepoID,
		Status:         models.CanvasRepositoryStatusReady,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := models.UpsertCanvasRepository(repository); err != nil {
		return nil, status.Error(codes.Internal, "failed to save canvas repository")
	}

	return repository, nil
}

func ensureWritableCanvasRepository(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*models.CanvasRepository, *models.Canvas, error) {
	repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, nil, err
	}

	canvas, err := models.FindCanvas(repository.OrganizationID, repository.CanvasID)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "failed to load canvas")
	}
	if canvas.IsTemplate {
		return nil, nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	return repository, canvas, nil
}

func canvasRepositoryCommitAuthor(ctx context.Context, organizationID string) (git.CommitAuthor, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return git.CommitAuthor{}, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	user, err := models.FindMaybeDeletedUserByID(organizationID, userID)
	if err != nil {
		return git.CommitAuthor{}, status.Error(codes.Internal, "failed to load user")
	}

	return git.CommitAuthor{
		Name:  user.Name,
		Email: canvasRepositoryCommitEmail(user),
	}, nil
}

func canvasRepositoryCommitEmail(user *models.User) string {
	if email := user.GetEmail(); email != "" {
		return email
	}

	return user.ID.String() + "@superplane.local"
}

func canvasRepositoryRef(repository *models.CanvasRepository) git.RepositoryRef {
	return git.RepositoryRef{
		RepoID:        repository.RepoID,
		DefaultBranch: "main",
	}
}

func RepositoryStateToProto(state string) pb.CanvasRepository_State {
	switch state {
	case models.CanvasRepositoryStatusPending:
		return pb.CanvasRepository_STATE_PENDING
	case models.CanvasRepositoryStatusReady:
		return pb.CanvasRepository_STATE_READY
	}
	return pb.CanvasRepository_STATE_UNSPECIFIED
}

func serializeCanvasRepository(ctx context.Context, repository *models.CanvasRepository, storage git.Provider) *pb.CanvasRepository {
	pbRepository := &pb.CanvasRepository{
		Metadata: &pb.CanvasRepository_Metadata{
			CanvasId:  repository.CanvasID.String(),
			UpdatedAt: timestamppb.New(repository.UpdatedAt),
		},
		Status: &pb.CanvasRepository_Status{
			State: RepositoryStateToProto(repository.Status),
		},
	}

	ref := canvasRepositoryRef(repository)
	head, err := storage.Head(ctx, ref, "main")
	if err == nil {
		pbRepository.Status.HeadSha = head
	}

	return pbRepository
}

func gitStorageStatusError(err error) error {
	switch {
	case errors.Is(err, git.ErrInvalidPath),
		errors.Is(err, git.ErrInvalidRepositoryID),
		errors.Is(err, git.ErrReservedPath),
		errors.Is(err, git.ErrInvalidCommit):
		return status.Errorf(codes.InvalidArgument, "%v", err)
	case errors.Is(err, git.ErrFileTooLarge),
		errors.Is(err, git.ErrCommitTooLarge):
		return status.Errorf(codes.ResourceExhausted, "%v", err)
	case errors.Is(err, git.ErrExpectedHeadMismatch):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	case errors.Is(err, git.ErrRemoteURLUnsupported):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}
