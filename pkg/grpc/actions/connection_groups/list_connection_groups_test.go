package connectiongroups

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListConnectionGroups(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := ListConnectionGroups(context.Background(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no connection groups", func(t *testing.T) {
		response, err := ListConnectionGroups(context.Background(), r.Canvas.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.ConnectionGroups)
	})

	t.Run("connection group exists", func(t *testing.T) {
		_, err := r.Canvas.CreateConnectionGroup(
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

		require.NoError(t, err)
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
