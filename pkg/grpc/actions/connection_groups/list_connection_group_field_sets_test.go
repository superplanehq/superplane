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

func Test__ListConnectionGroupFieldSets(t *testing.T) {
	r := support.Setup(t)

	connectionGroup, err := models.CreateConnectionGroup(
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
					{Name: "version", Expression: "ref"},
				},
			},
		},
	)

	require.NoError(t, err)

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := ListConnectionGroupFieldSets(context.Background(), uuid.NewString(), "test")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
	})

	t.Run("connection group does not exist -> error", func(t *testing.T) {
		_, err := ListConnectionGroupFieldSets(context.Background(), r.Canvas.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
	})

	t.Run("no field sets -> empty list", func(t *testing.T) {
		response, err := ListConnectionGroupFieldSets(context.Background(), r.Canvas.ID.String(), connectionGroup.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.FieldSets)
	})

	t.Run("field sets exist -> returns list", func(t *testing.T) {
		support.CreateFieldSet(t, map[string]string{"version": "v1"}, connectionGroup, r.Source)
		support.CreateFieldSet(t, map[string]string{"version": "v2"}, connectionGroup, r.Source)

		response, err := ListConnectionGroupFieldSets(context.Background(), r.Canvas.ID.String(), connectionGroup.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.FieldSets, 2)
	})
}
