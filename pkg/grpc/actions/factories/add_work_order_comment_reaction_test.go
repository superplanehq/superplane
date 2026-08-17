package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__AddWorkOrderCommentReaction(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	userIDStr := r.User.String()
	commentEvent, err := order.RecordCommentAdded(database.DB(t.Context()), "Nice", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil)
	require.NoError(t, err)

	t.Run("adds a reaction and returns the fresh summary", func(t *testing.T) {
		resp, err := AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "+1",
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.Equal(t, "+1", resp.Reactions[0].Emoji)
		assert.Equal(t, int32(1), resp.Reactions[0].Count)
		assert.True(t, resp.Reactions[0].ReactedByMe)
	})

	t.Run("adding the same reaction again is idempotent", func(t *testing.T) {
		resp, err := AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "+1",
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.Equal(t, int32(1), resp.Reactions[0].Count)
	})

	t.Run("rejects an emoji outside the fixed set", func(t *testing.T) {
		_, err := AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "not-a-real-emoji",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("not found comment", func(t *testing.T) {
		_, err := AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: uuid.New().String(),
			Emoji:     "eyes",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("not found factory", func(t *testing.T) {
		_, err := AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: uuid.New().String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "eyes",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := AddWorkOrderCommentReaction(context.Background(), r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "eyes",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})

	t.Run("a second user's reaction adds to the count without affecting reacted_by_me for the first user", func(t *testing.T) {
		other := support.CreateUser(t, r, r.Organization.ID)
		otherCtx := authentication.SetUserIdInMetadata(context.Background(), other.ID.String())

		resp, err := AddWorkOrderCommentReaction(otherCtx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "+1",
		})
		require.NoError(t, err)
		require.Len(t, resp.Reactions, 1)
		assert.Equal(t, int32(2), resp.Reactions[0].Count)
		assert.True(t, resp.Reactions[0].ReactedByMe)

		firstUserResp, err := AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "+1",
		})
		require.NoError(t, err)
		require.Len(t, firstUserResp.Reactions, 1)
		assert.Equal(t, int32(2), firstUserResp.Reactions[0].Count)
		assert.True(t, firstUserResp.Reactions[0].ReactedByMe)
	})
}

func Test__RemoveWorkOrderCommentReaction(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	userIDStr := r.User.String()
	commentEvent, err := order.RecordCommentAdded(database.DB(t.Context()), "Nice", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil)
	require.NoError(t, err)

	_, err = AddWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentReactionRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		CommentId: commentEvent.ID.String(),
		Emoji:     "heart",
	})
	require.NoError(t, err)

	t.Run("removes an existing reaction", func(t *testing.T) {
		resp, err := RemoveWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "heart",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Reactions)
	})

	t.Run("removing again is idempotent", func(t *testing.T) {
		resp, err := RemoveWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "heart",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Reactions)
	})

	t.Run("rejects an emoji outside the fixed set", func(t *testing.T) {
		_, err := RemoveWorkOrderCommentReaction(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "not-a-real-emoji",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := RemoveWorkOrderCommentReaction(context.Background(), r.Organization.ID.String(), &pb.RemoveWorkOrderCommentReactionRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			CommentId: commentEvent.ID.String(),
			Emoji:     "heart",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
