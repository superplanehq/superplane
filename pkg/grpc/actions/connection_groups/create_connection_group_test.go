package connectiongroups

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateConnectionGroup(t *testing.T) {
	r := support.Setup(t)

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: uuid.NewString(),
		}

		_, err := CreateConnectionGroup(ctx, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("no user ID in context -> error", func(t *testing.T) {
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			ConnectionGroup: &protos.ConnectionGroup{
				Metadata: &protos.ConnectionGroup_Metadata{
					Name: "test",
				},
			},
		}

		_, err := CreateConnectionGroup(context.Background(), req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Contains(t, s.Message(), "user not authenticated")
	})

	t.Run("connection group with no name -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			ConnectionGroup: &protos.ConnectionGroup{
				Metadata: &protos.ConnectionGroup_Metadata{},
			},
		}

		_, err := CreateConnectionGroup(ctx, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group name is required", s.Message())
	})

	t.Run("connection group with no connections -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			ConnectionGroup: &protos.ConnectionGroup{
				Metadata: &protos.ConnectionGroup_Metadata{
					Name: "test",
				},
				Spec: &protos.ConnectionGroup_Spec{
					Connections: []*protos.Connection{},
				},
			},
		}

		_, err := CreateConnectionGroup(ctx, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connections must not be empty", s.Message())
	})

	t.Run("connection group with no group by fields -> error", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			ConnectionGroup: &protos.ConnectionGroup{
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
			},
		}

		_, err := CreateConnectionGroup(ctx, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "connection group must have at least one field to group by", s.Message())
	})

	t.Run("valid connection group is created", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			ConnectionGroup: &protos.ConnectionGroup{
				Metadata: &protos.ConnectionGroup_Metadata{
					Name: "test",
				},
				Spec: &protos.ConnectionGroup_Spec{
					Connections: []*protos.Connection{
						{Name: r.Source.Name, Type: protos.Connection_TYPE_EVENT_SOURCE},
					},
					GroupBy: &protos.ConnectionGroup_Spec_GroupBy{
						Fields: []*protos.ConnectionGroup_Spec_GroupBy_Field{
							{Name: "test", Expression: "test"},
						},
					},
				},
			},
		}

		response, err := CreateConnectionGroup(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.ConnectionGroup)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.Id)
		assert.NotEmpty(t, response.ConnectionGroup.Metadata.CreatedAt)
		require.NotNil(t, response.ConnectionGroup.Spec)
		assert.Len(t, response.ConnectionGroup.Spec.Connections, 1)
		assert.Len(t, response.ConnectionGroup.Spec.GroupBy.Fields, 1)
	})

	t.Run("name already used", func(t *testing.T) {
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		req := &protos.CreateConnectionGroupRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			ConnectionGroup: &protos.ConnectionGroup{
				Metadata: &protos.ConnectionGroup_Metadata{
					Name: "test",
				},
				Spec: &protos.ConnectionGroup_Spec{
					Connections: []*protos.Connection{
						{Name: r.Source.Name, Type: protos.Connection_TYPE_EVENT_SOURCE},
					},
					GroupBy: &protos.ConnectionGroup_Spec_GroupBy{
						Fields: []*protos.ConnectionGroup_Spec_GroupBy_Field{
							{Name: "test", Expression: "test"},
						},
					},
				},
			},
		}

		_, err := CreateConnectionGroup(ctx, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
