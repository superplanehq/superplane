package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateFactory(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty name -> error", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "desc", "")
		require.NoError(t, err)

		emptyName := "   "
		_, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   factory.ID.String(),
			Name: &emptyName,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("updates factory metadata", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "old desc", "")
		require.NoError(t, err)

		newName := support.RandomName("updated-factory")
		newDescription := "Factory description updated"

		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:          factory.ID.String(),
			Name:        &newName,
			Description: &newDescription,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory)
		assert.Equal(t, factory.ID.String(), response.Factory.Id)
		assert.Equal(t, newName, response.Factory.Name)
		assert.Equal(t, newDescription, response.Factory.Description)

		updated, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		assert.Equal(t, newDescription, updated.Description)
	})

	t.Run("duplicate name -> error", func(t *testing.T) {
		existing, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		target, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   target.ID.String(),
			Name: &existing.Name,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
	})

	t.Run("not found -> error", func(t *testing.T) {
		name := support.RandomName("missing")
		_, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   "00000000-0000-0000-0000-000000000001",
			Name: &name,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("sets hosted spend limit", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		budget := int64(2500)
		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                     factory.ID.String(),
			HostedSpendBudgetCents: &budget,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory)
		require.NotNil(t, response.Factory.HostedSpendBudgetCents)
		assert.Equal(t, int64(2500), *response.Factory.HostedSpendBudgetCents)

		cleared, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                     factory.ID.String(),
			ClearHostedSpendBudget: true,
		})
		require.NoError(t, err)
		assert.Nil(t, cleared.Factory.HostedSpendBudgetCents)
	})
}
