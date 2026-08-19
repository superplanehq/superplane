package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__DeleteFactory(t *testing.T) {
	r := support.Setup(t)

	t.Run("factory is soft deleted and hidden from list/describe", func(t *testing.T) {
		originalName := support.RandomName("factory")
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, originalName, "desc", "")
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		require.NoError(t, err)

		_, err = models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		assert.ErrorIs(t, err, models.ErrFactoryNotFound)

		_, err = DescribeFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)

		listed, err := ListFactories(context.Background(), r.Organization.ID.String())
		require.NoError(t, err)
		for _, item := range listed.Factories {
			assert.NotEqual(t, factory.ID.String(), item.Id)
		}

		var deleted models.Factory
		err = database.DB(t.Context()).Unscoped().
			Where("organization_id = ? AND id = ?", r.Organization.ID, factory.ID).
			First(&deleted).
			Error
		require.NoError(t, err)
		assert.True(t, deleted.DeletedAt.Valid)
		assert.Contains(t, deleted.Name, "(deleted-")
		assert.NotEqual(t, originalName, deleted.Name)
	})

	t.Run("name can be reused after soft delete", func(t *testing.T) {
		name := support.RandomName("reuse-factory")
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, name, "", "")
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		require.NoError(t, err)

		recreated, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, name, "new", "")
		require.NoError(t, err)
		assert.Equal(t, name, recreated.Name)
		assert.NotEqual(t, factory.ID, recreated.ID)
	})

	t.Run("not found -> error", func(t *testing.T) {
		_, err := DeleteFactory(context.Background(), r.Organization.ID.String(), "00000000-0000-0000-0000-000000000001")
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("already deleted -> not found", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("soft deletes factory apps", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		require.NoError(t, database.DB(t.Context()).Model(canvas).Update("factory_id", factory.ID).Error)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		require.NoError(t, err)

		deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
		require.NoError(t, err)
		assert.True(t, deletedCanvas.DeletedAt.Valid)
		assert.Contains(t, deletedCanvas.Name, "(deleted-")
	})
}
