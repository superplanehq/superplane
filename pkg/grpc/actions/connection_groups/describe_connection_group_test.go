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

func Test__DescribeConnectionGroup(t *testing.T) {
	r := support.Setup(t)

	t.Run("connection group does not exist -> error", func(t *testing.T) {
		_, err := DescribeConnectionGroup(context.Background(), r.Canvas.ID.String(), uuid.NewString())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
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

		require.NoError(t, err)
		response, err := DescribeConnectionGroup(context.Background(), r.Canvas.ID.String(), "test")
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.ConnectionGroup)
		require.NotNil(t, response.ConnectionGroup.Metadata)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.Id)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.CreatedAt)
		require.NotNil(t, response.ConnectionGroup.Spec)
		assert.Len(t, response.ConnectionGroup.Spec.Connections, 1)
		assert.Len(t, response.ConnectionGroup.Spec.GroupBy.Fields, 1)
	})
}
