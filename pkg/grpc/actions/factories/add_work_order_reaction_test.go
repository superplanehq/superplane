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

func Test__AddWorkOrderReaction(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	otherCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("adds a reaction and reports reacted_by_me for the caller", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionThumbsUp,
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.Equal(t, models.ReactionThumbsUp, resp.Reactions[0].Content)
		assert.EqualValues(t, 1, resp.Reactions[0].Count)
		assert.True(t, resp.Reactions[0].ReactedByMe)
	})

	t.Run("reacting twice as the same user is idempotent", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionRocket,
		})
		require.NoError(t, err)

		resp, err := AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionRocket,
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.EqualValues(t, 1, resp.Reactions[0].Count, "reacting twice must not double-count")
	})

	t.Run("counts and reacted_by_me differ per caller", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionHeart,
		})
		require.NoError(t, err)

		resp, err := AddWorkOrderReaction(otherCtx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionHeart,
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.EqualValues(t, 2, resp.Reactions[0].Count)
		assert.True(t, resp.Reactions[0].ReactedByMe, "reacted_by_me reflects the calling user (otherUser)")
	})

	t.Run("rejects invalid content", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
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

		_, err = AddWorkOrderReaction(context.Background(), r.Organization.ID.String(), &pb.AddWorkOrderReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Content:   models.ReactionEyes,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
