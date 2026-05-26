package canvases

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvasstorage"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__DeleteCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, uuid.New().String(), nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, "invalid-id", nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas is soft deleted, data remains until cleanup", func(t *testing.T) {
		//
		// Create a canvas with nodes, events, and executions
		//
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
				{
					NodeID: "node-2",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		event1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		event2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)
		support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event1.ID, event2.ID, nil)
		support.CreateQueueItem(t, canvas.ID, "node-1", event1.ID, event2.ID)

		//
		// Verify canvas and all canvas data exist before deletion
		//
		_, err := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		nodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
		support.VerifyCanvasEventsCount(t, canvas.ID, 2)
		support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
		support.VerifyNodeQueueCount(t, canvas.ID, 1)

		//
		// Delete the canvas (soft delete).
		//
		_, err = DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, canvas.ID.String(), nil)
		require.NoError(t, err)

		//
		// Verify canvas is soft deleted but associated data still exists.
		// The canvas should not be found via regular queries (soft delete).
		//
		_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// But the workflow should still exist when queried with Unscoped.
		canvasUnscoped, err := models.FindUnscopedCanvas(canvas.ID)
		require.NoError(t, err)
		assert.NotNil(t, canvasUnscoped.DeletedAt)

		// Verify the name has been updated with deleted timestamp suffix
		assert.Contains(t, canvasUnscoped.Name, "(deleted-")
		assert.NotEqual(t, canvas.Name, canvasUnscoped.Name)

		// Associated data should still exist (cleanup worker handles this)
		nodes, err = models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		assert.Len(t, nodes, 2)
		support.VerifyCanvasEventsCount(t, canvas.ID, 2)
		support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
		support.VerifyNodeQueueCount(t, canvas.ID, 1)
	})

	t.Run("canvas node webhook remains until cleanup worker processes it", func(t *testing.T) {
		//
		// Create webhook
		//
		webhookID := uuid.New()
		webhook := models.Webhook{
			ID:     webhookID,
			State:  models.WebhookStatePending,
			Secret: []byte("secret"),
		}

		require.NoError(t, database.Conn().Create(&webhook).Error)

		//
		// Create a canvas with node that has webhook
		//
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: "node-1",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
					WebhookID: &webhookID,
				},
			},
			[]models.Edge{},
		)

		//
		// Delete the canvas (soft delete).
		//
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, canvas.ID.String(), nil)
		require.NoError(t, err)

		//
		// Verify canvas is soft deleted but webhook still exists.
		// The cleanup worker will handle webhook deletion later.
		//
		_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// Webhook should still exist since cleanup worker hasn't run
		_, err = models.FindWebhook(webhookID)
		require.NoError(t, err)
	})

	t.Run("deletes canvas repository before soft delete", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		repoID := canvasstorage.CanvasRepoID(r.Organization.ID, canvas.ID)
		repository := &models.CanvasRepository{
			CanvasID:       canvas.ID,
			OrganizationID: r.Organization.ID,
			Provider:       models.CanvasRepositoryProviderLocalGit,
			RepoID:         repoID,
			DefaultBranch:  "main",
			Status:         models.CanvasRepositoryStatusReady,
		}
		require.NoError(t, models.UpsertCanvasRepository(repository))

		storage := &deleteCanvasRepositoryStorage{}
		_, err := DeleteCanvas(context.Background(), r.Registry, r.Organization.ID, canvas.ID.String(), storage)
		require.NoError(t, err)

		require.Len(t, storage.deletedRefs, 1)
		assert.Equal(t, repoID, storage.deletedRefs[0].RepoID)
		assert.Equal(t, "main", storage.deletedRefs[0].DefaultBranch)
	})
}

type deleteCanvasRepositoryStorage struct {
	deletedRefs []canvasstorage.RepositoryRef
}

func (s *deleteCanvasRepositoryStorage) EnsureRepository(context.Context, canvasstorage.RepositorySpec) (*canvasstorage.Repository, error) {
	return nil, nil
}

func (s *deleteCanvasRepositoryStorage) DeleteRepository(_ context.Context, ref canvasstorage.RepositoryRef) error {
	s.deletedRefs = append(s.deletedRefs, ref)
	return nil
}

func (s *deleteCanvasRepositoryStorage) ListFiles(context.Context, canvasstorage.RepositoryRef, canvasstorage.ListFilesOptions) (*canvasstorage.ListFilesResult, error) {
	return nil, nil
}

func (s *deleteCanvasRepositoryStorage) GetFile(context.Context, canvasstorage.RepositoryRef, canvasstorage.GetFileOptions) (io.ReadCloser, error) {
	return nil, nil
}

func (s *deleteCanvasRepositoryStorage) CommitFiles(context.Context, canvasstorage.RepositoryRef, canvasstorage.CommitFilesOptions) (*canvasstorage.CommitResult, error) {
	return nil, nil
}

func (s *deleteCanvasRepositoryStorage) CurrentHead(context.Context, canvasstorage.RepositoryRef, string) (string, error) {
	return "", nil
}

func (s *deleteCanvasRepositoryStorage) GitURL(context.Context, canvasstorage.RepositoryRef) (string, error) {
	return "", nil
}

func (s *deleteCanvasRepositoryStorage) GenerateGitCredentials(context.Context, canvasstorage.RepositoryRef, canvasstorage.GitCredentialsOptions) (*canvasstorage.GitCredentials, error) {
	return nil, nil
}
