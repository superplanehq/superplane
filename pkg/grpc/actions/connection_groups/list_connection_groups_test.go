package connectiongroups

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__ListConnectionGroups(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.ListConnectionGroupsRequest{
			CanvasIdOrName: uuid.NewString(),
		}

		_, err := ListConnectionGroups(context.Background(), req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no connection groups", func(t *testing.T) {
		req := &protos.ListConnectionGroupsRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		}

		response, err := ListConnectionGroups(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Empty(t, response.ConnectionGroups)
	})

	t.Run("connection group exists", func(t *testing.T) {
		_, err := r.Canvas.CreateConnectionGroup(
			"test",
			uuid.NewString(),
			[]models.Connection{
				{SourceID: r.Source.ID, SourceName: r.Source.Name, SourceType: models.SourceTypeEventSource},
			},
			models.ConnectionGroupSpec{
				GroupBy: &models.ConnectionGroupBySpec{
					EmitOn: models.ConnectionGroupEmitOnAll,
					Fields: []models.ConnectionGroupByField{
						{Name: "test", Expression: "test"},
					},
				},
			},
		)

		req := &protos.ListConnectionGroupsRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
		}

		response, err := ListConnectionGroups(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.ConnectionGroups, 1)

		connectionGroup := response.ConnectionGroups[0]
		require.NotNil(t, connectionGroup.Metadata)
		assert.NotEmpty(t, connectionGroup.Metadata.Id)
		assert.NotEmpty(t, connectionGroup.Metadata.CreatedAt)
		require.NotNil(t, connectionGroup.Spec)
		assert.Len(t, connectionGroup.Spec.Connections, 1)
		assert.Equal(t, protos.ConnectionGroup_Spec_GroupBy_EMIT_ON_ALL, connectionGroup.Spec.GroupBy.EmitOn)
		assert.Len(t, connectionGroup.Spec.GroupBy.Fields, 1)
	})
}
