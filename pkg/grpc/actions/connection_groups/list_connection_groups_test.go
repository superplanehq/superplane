package connectiongroups

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListConnectionGroups(t *testing.T) {
	r := support.Setup(t)

	t.Run("no connection groups", func(t *testing.T) {
		response, err := ListConnectionGroups(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.ConnectionGroups)
	})

	t.Run("connection group exists", func(t *testing.T) {
		_, err := models.CreateConnectionGroup(
			r.Canvas.ID,
			"test",
			"test",
			uuid.NewString(),
			[]models.Connection{
				{SourceID: r.Source.ID, SourceName: r.Source.Name, SourceType: models.SourceTypeEventSource},
			},
			models.ConnectionGroupSpec{
				GroupBy: &models.ConnectionGroupBySpec{
					Fields: []models.ConnectionGroupByField{
						{Name: "test", Expression: "test"},
					},
				},
			},
		)

		response, err := ListConnectionGroups(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.ConnectionGroups, 1)

		connectionGroup := response.ConnectionGroups[0]
		require.NotNil(t, connectionGroup.Metadata)
		assert.NotEmpty(t, connectionGroup.Metadata.Id)
		assert.NotEmpty(t, connectionGroup.Metadata.CreatedAt)
		require.NotNil(t, connectionGroup.Spec)
		assert.Len(t, connectionGroup.Spec.Connections, 1)
		assert.Len(t, connectionGroup.Spec.GroupBy.Fields, 1)
	})
}
