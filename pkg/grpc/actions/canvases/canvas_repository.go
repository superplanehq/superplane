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
	"github.com/superplanehq/superplane/pkg/canvasstorage"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const maxCanvasRepositoryCredentialsTTL = 24 * time.Hour

type CanvasRepositoryStorageOptions struct {
	ProviderName  string
	DefaultBranch string
	MaxFileBytes  int64
}

func GetCanvasRepository(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage canvasstorage.Provider,
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
	ref string,
	storage canvasstorage.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.ListCanvasRepositoryFilesResponse, error) {
	repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	result, err := storage.ListFiles(ctx, canvasRepositoryRef(repository), canvasstorage.ListFilesOptions{Ref: ref})
	if err != nil {
		return nil, canvasStorageStatusError(err)
	}

	files := make([]*pb.CanvasRepositoryFile, 0, len(result.Paths))
	for _, path := range result.Paths {
		files = append(files, &pb.CanvasRepositoryFile{Path: path})
	}

	return &pb.ListCanvasRepositoryFilesResponse{
		Repository: serializeCanvasRepository(ctx, repository, storage),
		Files:      files,
		Ref:        result.Ref,
	}, nil
}

func GetCanvasRepositoryFile(
	ctx context.Context,
	organizationID string,
	canvasID string,
	path string,
	ref string,
	storage canvasstorage.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.GetCanvasRepositoryFileResponse, error) {
	repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	requestPath, err := url.PathUnescape(path)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file path")
	}

	normalizedPath, err := canvasstorage.NormalizePath(requestPath)
	if err != nil {
		return nil, canvasStorageStatusError(err)
	}

	reader, err := storage.GetFile(ctx, canvasRepositoryRef(repository), canvasstorage.GetFileOptions{
		Path: normalizedPath,
		Ref:  ref,
	})
	if err != nil {
		return nil, canvasStorageStatusError(err)
	}
	defer reader.Close()

	content, err := readCanvasRepositoryFile(reader, options.MaxFileBytes)
	if err != nil {
		return nil, err
	}

	return &pb.GetCanvasRepositoryFileResponse{
		Repository: serializeCanvasRepository(ctx, repository, storage),
		Path:       normalizedPath,
		Content:    content,
		Ref:        ref,
	}, nil
}

func CommitCanvasRepositoryFiles(
	ctx context.Context,
	organizationID string,
	canvasID string,
	req *pb.CommitCanvasRepositoryFilesRequest,
	storage canvasstorage.Provider,
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

	operations := make([]canvasstorage.FileOperation, 0, len(req.GetOperations()))
	for _, operation := range req.GetOperations() {
		content := operation.GetContent()
		var reader io.Reader
		if !operation.GetDelete() {
			reader = bytes.NewReader(content)
		}

		operations = append(operations, canvasstorage.FileOperation{
			Path:      operation.GetPath(),
			Content:   reader,
			SizeBytes: int64(len(content)),
			Delete:    operation.GetDelete(),
		})
	}

	result, err := storage.CommitFiles(ctx, canvasRepositoryRef(repository), canvasstorage.CommitFilesOptions{
		Branch:          req.GetBranch(),
		BaseBranch:      req.GetBaseBranch(),
		ExpectedHeadSHA: req.GetExpectedHeadSha(),
		Message:         req.GetMessage(),
		Author:          author,
		Operations:      operations,
	})
	if err != nil {
		return nil, canvasStorageStatusError(err)
	}

	if result.NewSHA != "" {
		now := time.Now()
		repository.Status = models.CanvasRepositoryStatusReady
		repository.UpdatedAt = now
		if err := models.UpsertCanvasRepository(repository); err != nil {
			return nil, status.Error(codes.Internal, "failed to update canvas repository")
		}
	}

	return &pb.CommitCanvasRepositoryFilesResponse{
		Repository: serializeCanvasRepository(ctx, repository, storage),
		CommitSha:  result.CommitSHA,
		OldSha:     result.OldSHA,
		NewSha:     result.NewSHA,
		Branch:     result.Branch,
	}, nil
}

func GenerateCanvasRepositoryCredentials(
	ctx context.Context,
	organizationID string,
	canvasID string,
	req *pb.GenerateCanvasRepositoryCredentialsRequest,
	storage canvasstorage.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.GenerateCanvasRepositoryCredentialsResponse, error) {
	if req.GetReadOnly() {
		repository, err := ensureCanvasRepository(ctx, organizationID, canvasID, storage, options)
		if err != nil {
			return nil, err
		}
		return generateCanvasRepositoryCredentials(ctx, repository, req, storage)
	}

	repository, _, err := ensureWritableCanvasRepository(ctx, organizationID, canvasID, storage, options)
	if err != nil {
		return nil, err
	}

	return generateCanvasRepositoryCredentials(ctx, repository, req, storage)
}

func generateCanvasRepositoryCredentials(
	ctx context.Context,
	repository *models.CanvasRepository,
	req *pb.GenerateCanvasRepositoryCredentialsRequest,
	storage canvasstorage.Provider,
) (*pb.GenerateCanvasRepositoryCredentialsResponse, error) {
	ttl := time.Duration(req.GetTtlSeconds()) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	if ttl > maxCanvasRepositoryCredentialsTTL {
		ttl = maxCanvasRepositoryCredentialsTTL
	}

	credentials, err := storage.GenerateGitCredentials(ctx, canvasRepositoryRef(repository), canvasstorage.GitCredentialsOptions{
		ReadOnly:       req.GetReadOnly(),
		TTL:            ttl,
		AllowForcePush: req.GetAllowForcePush(),
	})
	if err != nil {
		return nil, canvasStorageStatusError(err)
	}

	return &pb.GenerateCanvasRepositoryCredentialsResponse{
		Repository: serializeCanvasRepository(ctx, repository, storage),
		Username:   credentials.Username,
		Password:   credentials.Password,
		ExpiresAt:  timestamppb.New(time.Now().Add(ttl)),
	}, nil
}

func ensureCanvasRepository(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage canvasstorage.Provider,
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

	storageRepo, err := storage.EnsureRepository(ctx, canvasstorage.RepositorySpec{
		OrganizationID: organizationUUID,
		CanvasID:       canvas.ID,
		RepoID:         canvasstorage.CanvasRepoID(organizationUUID, canvas.ID),
		DefaultBranch:  options.DefaultBranch,
	})
	if err != nil {
		return nil, canvasStorageStatusError(err)
	}

	now := time.Now()
	repository = &models.CanvasRepository{
		CanvasID:       canvas.ID,
		OrganizationID: organizationUUID,
		Provider:       options.ProviderName,
		RepoID:         storageRepo.RepoID,
		DefaultBranch:  storageRepo.DefaultBranch,
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
	storage canvasstorage.Provider,
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

func canvasRepositoryCommitAuthor(ctx context.Context, organizationID string) (canvasstorage.CommitAuthor, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return canvasstorage.CommitAuthor{}, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	user, err := models.FindMaybeDeletedUserByID(organizationID, userID)
	if err != nil {
		return canvasstorage.CommitAuthor{}, status.Error(codes.Internal, "failed to load user")
	}

	return canvasstorage.CommitAuthor{
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

func canvasRepositoryRef(repository *models.CanvasRepository) canvasstorage.RepositoryRef {
	return canvasstorage.RepositoryRef{
		RepoID:        repository.RepoID,
		DefaultBranch: repository.DefaultBranch,
	}
}

func serializeCanvasRepository(ctx context.Context, repository *models.CanvasRepository, storage canvasstorage.Provider) *pb.CanvasRepository {
	pbRepository := &pb.CanvasRepository{
		CanvasId:      repository.CanvasID.String(),
		Provider:      repository.Provider,
		RepoId:        repository.RepoID,
		DefaultBranch: repository.DefaultBranch,
		Status:        repository.Status,
		UpdatedAt:     timestamppb.New(repository.UpdatedAt),
	}

	if storage != nil {
		ref := canvasRepositoryRef(repository)
		head, err := storage.CurrentHead(ctx, ref, repository.DefaultBranch)
		if err == nil {
			pbRepository.HeadSha = head
		}

		gitURL, err := storage.GitURL(ctx, ref)
		if err == nil {
			pbRepository.GitUrl = gitURL
		}
	}

	return pbRepository
}

func readCanvasRepositoryFile(reader io.Reader, maxFileBytes int64) ([]byte, error) {
	if maxFileBytes <= 0 {
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to read file")
		}
		return content, nil
	}

	limited := io.LimitReader(reader, maxFileBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read file")
	}
	if int64(len(content)) > maxFileBytes {
		return nil, status.Error(codes.ResourceExhausted, "file exceeds configured size limit")
	}

	return content, nil
}

func canvasStorageStatusError(err error) error {
	switch {
	case errors.Is(err, canvasstorage.ErrInvalidPath),
		errors.Is(err, canvasstorage.ErrInvalidRepositoryID),
		errors.Is(err, canvasstorage.ErrReservedPath),
		errors.Is(err, canvasstorage.ErrInvalidCommit):
		return status.Errorf(codes.InvalidArgument, "%v", err)
	case errors.Is(err, canvasstorage.ErrFileTooLarge),
		errors.Is(err, canvasstorage.ErrCommitTooLarge):
		return status.Errorf(codes.ResourceExhausted, "%v", err)
	case errors.Is(err, canvasstorage.ErrExpectedHeadMismatch):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	case errors.Is(err, canvasstorage.ErrRemoteURLUnsupported):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}
