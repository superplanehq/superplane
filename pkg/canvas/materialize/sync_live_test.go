package materialize_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const syncLiveWebhookBaseURL = "http://localhost:3000/api/v1"

func TestSyncLiveFromGit(t *testing.T) {
	r := support.Setup(t)

	t.Run("rejects external sync when change management enabled", func(t *testing.T) {
		canvas, _ := support.CreateCanvasGitFirst(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{},
			[]models.Edge{},
		)
		require.NotNil(t, canvas.LiveVersionID)
		headSHA := *canvas.LiveVersionID

		require.NoError(t, database.Conn().Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("change_management_enabled", true).Error)

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			_, syncErr := materialize.SyncLiveFromGit(
				context.Background(),
				tx,
				r.GitProvider,
				r.Registry,
				r.Encryptor,
				r.AuthService,
				syncLiveWebhookBaseURL,
				r.Organization.ID,
				canvas.ID,
				materialize.SyncLiveFromGitOptions{HeadSHA: headSHA},
			)
			return syncErr
		})
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())

		state, stateErr := models.FindRepositoryMaterializationState(canvas.ID, models.CanvasGitBranchMain)
		require.NoError(t, stateErr)
		assert.Equal(t, models.MaterializationStatusError, state.Status)

		require.NoError(t, database.Conn().Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("change_management_enabled", false).Error)
	})

	t.Run("idempotent when main already materialized", func(t *testing.T) {
		canvas, _ := support.CreateCanvasGitFirst(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{},
			[]models.Edge{},
		)
		require.NotNil(t, canvas.LiveVersionID)
		initialSHA := *canvas.LiveVersionID

		var version *models.CanvasVersion
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			var syncErr error
			version, syncErr = materialize.SyncLiveFromGit(
				context.Background(),
				tx,
				r.GitProvider,
				r.Registry,
				r.Encryptor,
				r.AuthService,
				syncLiveWebhookBaseURL,
				r.Organization.ID,
				canvas.ID,
				materialize.SyncLiveFromGitOptions{HeadSHA: initialSHA},
			)
			return syncErr
		})
		require.NoError(t, err)
		require.NotNil(t, version)
		assert.Equal(t, initialSHA, version.ID)
	})

	t.Run("allows sync when change management check skipped", func(t *testing.T) {
		canvas, _ := support.CreateCanvasGitFirst(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{},
			[]models.Edge{},
		)
		require.NotNil(t, canvas.LiveVersionID)
		headSHA := *canvas.LiveVersionID

		require.NoError(t, database.Conn().Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("change_management_enabled", true).Error)

		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			_, syncErr := materialize.SyncLiveFromGit(
				context.Background(),
				tx,
				r.GitProvider,
				r.Registry,
				r.Encryptor,
				r.AuthService,
				syncLiveWebhookBaseURL,
				r.Organization.ID,
				canvas.ID,
				materialize.SyncLiveFromGitOptions{
					HeadSHA:                   headSHA,
					SkipChangeManagementCheck: true,
				},
			)
			return syncErr
		})
		require.NoError(t, err)

		require.NoError(t, database.Conn().Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Update("change_management_enabled", false).Error)
	})
}
