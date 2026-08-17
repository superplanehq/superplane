package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__RemoveWorkOrderReaction(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("removes an existing reaction", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionThumbsUp,
		})
		require.NoError(t, err)

		resp, err := RemoveWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionThumbsUp,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Reactions)
	})

	t.Run("removing a reaction that was never added is a no-op", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := RemoveWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionLaugh,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Reactions)
	})

	t.Run("removing one reaction doesn't affect the user's other reactions", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionThumbsUp,
		})
		require.NoError(t, err)
		_, err = AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionEyes,
		})
		require.NoError(t, err)

		resp, err := RemoveWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionThumbsUp,
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.Equal(t, models.ReactionEyes, resp.Reactions[0].Content)
	})

	t.Run("rejects invalid content", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = RemoveWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   "not-a-real-emoji",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = RemoveWorkOrderReaction(context.Background(), r.Organization.ID.String(), &pb.RemoveWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionEyes,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
