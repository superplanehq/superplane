package connectiongroups

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateConnectionGroup(t *testing.T) {
	r := support.Setup(t)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

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
					{Name: "test", Expression: "test"},
				},
			},
		},
	)

	require.NoError(t, err)

	t.Run("wrong canvas -> error", func(t *testing.T) {
		_, err := UpdateConnectionGroup(ctx, uuid.NewString(), connectionGroup.ID.String(), &protos.ConnectionGroup{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
	})

	t.Run("connection group does not exist -> error", func(t *testing.T) {
		_, err := UpdateConnectionGroup(ctx, r.Canvas.ID.String(), uuid.NewString(), &protos.ConnectionGroup{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group not found", s.Message())
	})

	t.Run("no user ID in context -> error", func(t *testing.T) {
		_, err := UpdateConnectionGroup(context.Background(), r.Canvas.ID.String(), connectionGroup.ID.String(), &protos.ConnectionGroup{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Contains(t, s.Message(), "user not authenticated")
	})

	t.Run("connection group with no connections -> error", func(t *testing.T) {
		_, err := UpdateConnectionGroup(ctx, r.Canvas.ID.String(), connectionGroup.ID.String(), &protos.ConnectionGroup{
			Metadata: &protos.ConnectionGroup_Metadata{
				Name: "test",
			},
			Spec: &protos.ConnectionGroup_Spec{
				Connections: []*protos.Connection{},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connections must not be empty", s.Message())
	})

	t.Run("connection group with no group by fields -> error", func(t *testing.T) {
		_, err := UpdateConnectionGroup(ctx, r.Canvas.ID.String(), connectionGroup.ID.String(), &protos.ConnectionGroup{
			Metadata: &protos.ConnectionGroup_Metadata{
				Name: "test",
			},
			Spec: &protos.ConnectionGroup_Spec{
				Connections: []*protos.Connection{
					{Name: r.Source.Name, Type: protos.Connection_TYPE_EVENT_SOURCE},
				},
				GroupBy: &protos.ConnectionGroup_Spec_GroupBy{
					Fields: []*protos.ConnectionGroup_Spec_GroupBy_Field{},
				},
			},
		})

		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group must have at least one field to group by", s.Message())
	})

	t.Run("connection group is updated", func(t *testing.T) {
		response, err := UpdateConnectionGroup(ctx, r.Canvas.ID.String(), connectionGroup.ID.String(), &protos.ConnectionGroup{
			Metadata: &protos.ConnectionGroup_Metadata{
				Name:        "updated-test",
				Description: "updated-description",
			},
			Spec: &protos.ConnectionGroup_Spec{
				Connections: []*protos.Connection{
					{Name: r.Source.Name, Type: protos.Connection_TYPE_EVENT_SOURCE},
					{Name: r.Stage.Name, Type: protos.Connection_TYPE_STAGE},
				},
				GroupBy: &protos.ConnectionGroup_Spec_GroupBy{
					Fields: []*protos.ConnectionGroup_Spec_GroupBy_Field{
						{Name: "a", Expression: "a"},
						{Name: "b", Expression: "b"},
					},
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.ConnectionGroup)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.Id)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.CreatedAt)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.UpdatedAt)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.UpdatedBy)
		require.NotNil(t, response.ConnectionGroup.Spec)
		assert.Equal(t, "updated-test", response.ConnectionGroup.Metadata.Name)
		assert.Equal(t, "updated-description", response.ConnectionGroup.Metadata.Description)
		assert.Len(t, response.ConnectionGroup.Spec.Connections, 2)
		assert.Len(t, response.ConnectionGroup.Spec.GroupBy.Fields, 2)
	})
}
