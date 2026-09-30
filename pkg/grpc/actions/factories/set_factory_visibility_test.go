package factories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__SetFactoryVisibility(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("rejects a workspace that is still in setup", func(t *testing.T) {
		factory := newVisibilityFactory(t, r)
		_, err := SetFactoryVisibility(ctx, r.Organization.ID.String(), &pb.SetFactoryVisibilityRequest{
			Id:     factory.ID.String(),
			Public: true,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("admin path sets public and update does not clear it", func(t *testing.T) {
		factory := newVisibilityFactory(t, r)
		finishFactoryOnboarding(t, factory)

		response, err := SetFactoryVisibility(ctx, r.Organization.ID.String(), &pb.SetFactoryVisibilityRequest{
			Id:     factory.ID.String(),
			Public: true,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory)
		assert.True(t, response.Factory.Public)

		name := support.RandomName("renamed-factory")
		_, err = UpdateFactory(ctx, r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   factory.ID.String(),
			Name: &name,
		})
		require.NoError(t, err)

		updated, err := models.FindFactory(database.Conn(), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Equal(t, name, updated.Name)
		assert.True(t, updated.Public)
	})
}

func newVisibilityFactory(t *testing.T, r *support.ResourceRegistry) *models.Factory {
	t.Helper()
	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "desc", "")
	require.NoError(t, err)
	return factory
}

func finishFactoryOnboarding(t *testing.T, factory *models.Factory) {
	t.Helper()
	now := time.Now()
	require.NoError(t, database.Conn().Model(factory).Update("onboarding_completed_at", now).Error)
	factory.OnboardingCompletedAt = &now
}
