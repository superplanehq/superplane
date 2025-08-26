package connectiongroups

import (
	"context"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func Test__DeleteConnectionGroup(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})

	t.Run("wrong canvas -> error", func(t *testing.T) {
		spec := models.ConnectionGroupSpec{
			GroupBy: &models.ConnectionGroupBySpec{
				Fields: []models.ConnectionGroupByField{
					{Name: "field1", Expression: "$.test"},
				},
			},
			Timeout:         300,
			TimeoutBehavior: models.ConnectionGroupTimeoutBehaviorNone,
		}

		connectionGroup, err := models.CreateConnectionGroup(
			r.Canvas.ID,
			"test-connection-group-wrong-canvas",
			"Test Connection Group",
			r.User.String(),
			[]models.Connection{},
			spec,
		)
		require.NoError(t, err)

		_, err = DeleteConnectionGroup(context.Background(), uuid.NewString(), connectionGroup.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
	})

	t.Run("connection group that does not exist -> error", func(t *testing.T) {
		_, err := DeleteConnectionGroup(context.Background(), r.Canvas.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
	})

	t.Run("delete connection group by id successfully", func(t *testing.T) {
		spec := models.ConnectionGroupSpec{
			GroupBy: &models.ConnectionGroupBySpec{
				Fields: []models.ConnectionGroupByField{
					{Name: "field1", Expression: "$.test"},
				},
			},
			Timeout:         300,
			TimeoutBehavior: models.ConnectionGroupTimeoutBehaviorNone,
		}

		connectionGroup, err := models.CreateConnectionGroup(
			r.Canvas.ID,
			"test-connection-group-by-id",
			"Test Connection Group By ID",
			r.User.String(),
			[]models.Connection{},
			spec,
		)
		require.NoError(t, err)

		response, err := DeleteConnectionGroup(context.Background(), r.Canvas.ID.String(), connectionGroup.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindConnectionGroupByID(r.Canvas.ID.String(), connectionGroup.ID.String())
		assert.Error(t, err)
		assert.True(t, err == gorm.ErrRecordNotFound)

		softDeletedGroups, err := models.ListUnscopedSoftDeletedConnectionGroups(10, time.Now().Add(time.Hour))
		require.NoError(t, err)
		found := false
		for _, g := range softDeletedGroups {
			if g.ID == connectionGroup.ID {
				found = true
				assert.Contains(t, g.Name, "deleted-")
				break
			}
		}
		assert.True(t, found, "Connection group should be in soft deleted list")
	})

	t.Run("delete connection group by name successfully", func(t *testing.T) {
		spec := models.ConnectionGroupSpec{
			GroupBy: &models.ConnectionGroupBySpec{
				Fields: []models.ConnectionGroupByField{
					{Name: "field1", Expression: "$.test"},
				},
			},
			Timeout:         300,
			TimeoutBehavior: models.ConnectionGroupTimeoutBehaviorNone,
		}

		connectionGroup, err := models.CreateConnectionGroup(
			r.Canvas.ID,
			"test-connection-group-by-name",
			"Test Connection Group By Name",
			r.User.String(),
			[]models.Connection{},
			spec,
		)
		require.NoError(t, err)

		response, err := DeleteConnectionGroup(context.Background(), r.Canvas.ID.String(), connectionGroup.Name)
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindConnectionGroupByName(r.Canvas.ID.String(), connectionGroup.Name)
		assert.Error(t, err)
		assert.True(t, err == gorm.ErrRecordNotFound)
	})
}
