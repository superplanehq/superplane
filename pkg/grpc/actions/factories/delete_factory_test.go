package factories

import (
	"context"
	"fmt"
	"strings"
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
		recorder := useAnalytics(t)
		originalName := support.RandomName("factory")
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, originalName, "desc", "")
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		require.NoError(t, err)

		_, err = models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		assert.ErrorIs(t, err, models.ErrFactoryNotFound)

		_, err = DescribeFactory(context.Background(), IntakeDependencies{}, r.Organization.ID.String(), factory.ID.String())
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
		events := recorder.snapshot()
		require.Len(t, events, 1)
		assert.Equal(t, "factory_deleted", string(events[0].Name))
		assert.Equal(t, factory.ID.String(), events[0].FactoryID)
		assert.Equal(t, r.Organization.ID.String(), events[0].OrganizationID)
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
		recorder := useAnalytics(t)
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		require.NoError(t, err)

		_, err = DeleteFactory(context.Background(), r.Organization.ID.String(), factory.ID.String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		require.Len(t, recorder.snapshot(), 1)
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

	t.Run("rollback preserves factory apps and emits nothing", func(t *testing.T) {
		db := database.DB(t.Context())
		recorder := useAnalytics(t)
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, support.RandomName("app"))
		name := "reject_factory_delete_" + strings.ReplaceAll(factory.ID.String(), "-", "")
		require.NoError(t, db.Exec(fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.id = '%s'::uuid THEN RAISE EXCEPTION 'factory deletion failure'; END IF; RETURN NEW; END $$`, name, factory.ID)).Error)
		t.Cleanup(func() {
			require.NoError(t, database.DB(context.WithoutCancel(t.Context())).Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS %s() CASCADE", name)).Error)
		})
		require.NoError(t, db.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE ON factories FOR EACH ROW EXECUTE FUNCTION %s()`, name, name)).Error)
		_, err = DeleteFactory(t.Context(), r.Organization.ID.String(), factory.ID.String())
		require.Error(t, err)
		require.Empty(t, recorder.snapshot())
		_, err = models.FindFactory(db, r.Organization.ID, factory.ID)
		require.NoError(t, err)
		stored, err := models.FindUnscopedCanvas(canvas.ID)
		require.NoError(t, err)
		require.False(t, stored.DeletedAt.Valid)
	})
}
