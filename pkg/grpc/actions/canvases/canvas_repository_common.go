package canvases

import (
	"context"
	"errors"

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

func loadCanvasRepository(
	organizationID string,
	canvasID string,
) (*models.CanvasRepository, error) {
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

	if canvas.IsTemplate {
		return nil, status.Error(codes.NotFound, "canvas repository not found")
	}

	repository, err := models.FindCanvasRepository(canvas.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas repository not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas repository")
	}

	if repository.OrganizationID != organizationUUID {
		return nil, status.Error(codes.NotFound, "canvas repository not found")
	}

	return repository, nil
}

func requireReadyCanvasRepository(
	organizationID string,
	canvasID string,
	storage git.Provider,
) (*models.CanvasRepository, error) {
	if storage == nil {
		return nil, status.Error(codes.FailedPrecondition, "canvas file storage is not configured")
	}

	repository, err := loadCanvasRepository(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	switch repository.Status {
	case models.CanvasRepositoryStatusReady:
		return repository, nil
	case models.CanvasRepositoryStatusPending, models.CanvasRepositoryStatusProvisioning:
		return nil, status.Error(codes.FailedPrecondition, "canvas repository is not ready")
	case models.CanvasRepositoryStatusError:
		return nil, status.Error(codes.FailedPrecondition, "canvas repository failed to provision")
	default:
		return nil, status.Error(codes.FailedPrecondition, "canvas repository is not ready")
	}
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
	case models.CanvasRepositoryStatusPending, models.CanvasRepositoryStatusProvisioning:
		return pb.CanvasRepository_STATE_PENDING
	case models.CanvasRepositoryStatusReady:
		return pb.CanvasRepository_STATE_READY
	case models.CanvasRepositoryStatusError:
		return pb.CanvasRepository_STATE_ERROR
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

	if repository.Status != models.CanvasRepositoryStatusReady || storage == nil {
		return pbRepository
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
