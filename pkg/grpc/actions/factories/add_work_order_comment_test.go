package factories

import (
	"context"
	"encoding/json"
	"testing"

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

func Test__AddWorkOrderComment(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	userA := support.CreateUser(t, r, r.Organization.ID)
	userB := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("stores and returns mentioned users, deduped", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := AddWorkOrderComment(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentRequest{
			FactoryId:        factoryModel.ID.String(),
			OrderId:          order.ID.String(),
			Body:             "Hey @userA @userB, take a look",
			MentionedUserIds: []string{userA.ID.String(), userB.ID.String(), userA.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, resp.Comment.Mentions, 2)

		mentionedIDs := []string{resp.Comment.Mentions[0].Id, resp.Comment.Mentions[1].Id}
		assert.ElementsMatch(t, []string{userA.ID.String(), userB.ID.String()}, mentionedIDs)

		refreshed, err := factoryModel.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		comments, err := refreshed.ListComments(database.DB(t.Context()))
		require.NoError(t, err)
		require.Len(t, comments, 1)

		var payload factory.WorkOrderCommentAdded
		require.NoError(t, json.Unmarshal(comments[0].Data, &payload))
		require.Len(t, payload.Mentions, 2)
	})

	t.Run("drops a self-mention", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := AddWorkOrderComment(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentRequest{
			FactoryId:        factoryModel.ID.String(),
			OrderId:          order.ID.String(),
			Body:             "Noting this myself",
			MentionedUserIds: []string{r.User.String(), userA.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, resp.Comment.Mentions, 1)
		assert.Equal(t, userA.ID.String(), resp.Comment.Mentions[0].Id)
	})

	t.Run("rejects unknown mentioned user id", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderComment(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentRequest{
			FactoryId:        factoryModel.ID.String(),
			OrderId:          order.ID.String(),
			Body:             "cc someone who doesn't exist",
			MentionedUserIds: []string{"00000000-0000-0000-0000-000000000099"},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a malformed mentioned user id", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = AddWorkOrderComment(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentRequest{
			FactoryId:        factoryModel.ID.String(),
			OrderId:          order.ID.String(),
			Body:             "cc not-a-uuid",
			MentionedUserIds: []string{"not-a-uuid"},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("comments without mentions have an empty mentions list", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := AddWorkOrderComment(ctx, r.Organization.ID.String(), &pb.AddWorkOrderCommentRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Body:      "No mentions here",
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Comment.Mentions)
	})
}
