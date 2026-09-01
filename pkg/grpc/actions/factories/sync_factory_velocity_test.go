package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SyncFactoryVelocity(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())

	newSyncableFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		integrationID := uuid.NewString()
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID,
			AppRepository:    &appRepo,
		}))
		return factory
	}

	t.Run("hands the sync to the worker and returns before it runs", func(t *testing.T) {
		factory := newSyncableFactory(t)

		resp, err := SyncFactoryVelocity(context.Background(), orgID, &pb.SyncFactoryVelocityRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.True(t, resp.GetStarted())
	})

	t.Run("a workspace with no repository has nothing to sync", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		resp, err := SyncFactoryVelocity(context.Background(), orgID, &pb.SyncFactoryVelocityRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		assert.False(t, resp.GetStarted(), "the UI explains the gap instead of showing endless progress")
	})

	t.Run("a workspace of another organization is not found", func(t *testing.T) {
		factory := newSyncableFactory(t)
		otherOrg := support.CreateOrganization(t, r, r.User)

		_, err := SyncFactoryVelocity(context.Background(), otherOrg.ID.String(),
			&pb.SyncFactoryVelocityRequest{FactoryId: factory.ID.String()})

		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})
}
