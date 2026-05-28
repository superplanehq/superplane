package workers

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/git"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type fakeCanvasRepositoryStorage struct {
	createCalls int
	initCalls   int
}

func (s *fakeCanvasRepositoryStorage) CreateRepository(context.Context, git.RepositorySpec) (*git.Repository, error) {
	s.createCalls++
	return &git.Repository{RepoID: "test-repo", DefaultBranch: "main"}, nil
}

func (s *fakeCanvasRepositoryStorage) DeleteRepository(context.Context, git.RepositoryRef) error {
	return nil
}

func (s *fakeCanvasRepositoryStorage) ListFiles(context.Context, git.RepositoryRef, git.ListFilesOptions) (*git.ListFilesResult, error) {
	return &git.ListFilesResult{}, nil
}

func (s *fakeCanvasRepositoryStorage) GetFile(context.Context, git.RepositoryRef, git.GetFileOptions) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

func (s *fakeCanvasRepositoryStorage) InitRepository(context.Context, git.RepositoryRef, string) error {
	s.initCalls++
	return nil
}

func (s *fakeCanvasRepositoryStorage) Commit(context.Context, git.RepositoryRef, git.CommitOptions) (*git.CommitResult, error) {
	return &git.CommitResult{CommitSHA: "abc123"}, nil
}

func (s *fakeCanvasRepositoryStorage) Head(context.Context, git.RepositoryRef, string) (string, error) {
	return "abc123", nil
}

func Test__CanvasRepositoryProvisionerWorker_ProvisionsPendingRepository(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	storage := &fakeCanvasRepositoryStorage{}
	options := canvases.CanvasRepositoryStorageOptions{
		ProviderName:  models.CanvasRepositoryProviderSupergit,
		DefaultBranch: "main",
	}

	response, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		"https://example.com",
		r.Organization.ID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Files Canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		nil,
		&options,
	)
	require.NoError(t, err)

	canvasID := uuid.MustParse(response.Canvas.Metadata.Id)
	repository, err := models.FindCanvasRepository(canvasID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasRepositoryStatusPending, repository.Status)

	worker := NewCanvasRepositoryProvisionerWorker("amqp://unused", storage, options)
	require.NoError(t, worker.provisionRepository(context.Background(), canvasID))
	require.Equal(t, 1, storage.createCalls)
	require.Equal(t, 1, storage.initCalls)

	repository, err = models.FindCanvasRepository(canvasID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasRepositoryStatusReady, repository.Status)
}

func Test__CreateCanvas_CreatesPendingCanvasRepositoryWhenStorageEnabled(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	options := canvases.CanvasRepositoryStorageOptions{
		ProviderName:  models.CanvasRepositoryProviderSupergit,
		DefaultBranch: "main",
	}

	response, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		"https://example.com",
		r.Organization.ID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "Pending Repo Canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		nil,
		&options,
	)
	require.NoError(t, err)

	canvasID := uuid.MustParse(response.Canvas.Metadata.Id)
	repository, err := models.FindCanvasRepository(canvasID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasRepositoryStatusPending, repository.Status)
	require.Equal(t, options.ProviderName, repository.Provider)
	require.Equal(t, git.CanvasRepoID(r.Organization.ID, canvasID), repository.RepoID)
	require.WithinDuration(t, time.Now(), repository.CreatedAt, 5*time.Second)
}

func Test__CreateCanvas_SkipsPendingCanvasRepositoryWhenStorageDisabled(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	response, err := canvases.CreateCanvas(
		ctx,
		r.Registry,
		r.Encryptor,
		r.AuthService,
		"https://example.com",
		r.Organization.ID,
		&pb.Canvas{
			Metadata: &pb.Canvas_Metadata{Name: "No Repo Canvas"},
			Spec: &pb.Canvas_Spec{
				Nodes: []*componentpb.Node{},
				Edges: []*componentpb.Edge{},
			},
		},
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)

	canvasID := uuid.MustParse(response.Canvas.Metadata.Id)
	_, err = models.FindCanvasRepository(canvasID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
