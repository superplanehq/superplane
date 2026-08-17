package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestFactoryWorkOrder_AddCommentReaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, userID, factoryModel := setupFactoryWithUser(t, "reaction-add")
	otherUserID := createSecondUser(t, org, "reaction-add-other")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Reaction target", "", &userID, nil, nil)
	require.NoError(t, err)

	userIDStr := userID.String()
	commentEvent, err := order.RecordCommentAdded(database.Conn(), "Nice work", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil)
	require.NoError(t, err)

	t.Run("rejects an unknown emoji at the model layer via IsValidCommentReactionEmoji", func(t *testing.T) {
		assert.False(t, IsValidCommentReactionEmoji("not-an-emoji"))
		assert.True(t, IsValidCommentReactionEmoji(CommentReactionRocket))
	})

	t.Run("adding a reaction is reflected in the summary", func(t *testing.T) {
		require.NoError(t, order.AddCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionThumbsUp))

		summaries, err := ListCommentReactionSummaries(database.Conn(), order.ID, []uuid.UUID{commentEvent.ID}, userID)
		require.NoError(t, err)
		require.Len(t, summaries[commentEvent.ID], 1)
		assert.Equal(t, CommentReactionThumbsUp, summaries[commentEvent.ID][0].Emoji)
		assert.Equal(t, 1, summaries[commentEvent.ID][0].Count)
		assert.True(t, summaries[commentEvent.ID][0].ReactedByMe)
	})

	t.Run("adding the same reaction twice is idempotent", func(t *testing.T) {
		require.NoError(t, order.AddCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionThumbsUp))
		require.NoError(t, order.AddCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionThumbsUp))

		summaries, err := ListCommentReactionSummaries(database.Conn(), order.ID, []uuid.UUID{commentEvent.ID}, userID)
		require.NoError(t, err)
		require.Len(t, summaries[commentEvent.ID], 1)
		assert.Equal(t, 1, summaries[commentEvent.ID][0].Count)
	})

	t.Run("counts aggregate across users and reacted_by_me is per caller", func(t *testing.T) {
		require.NoError(t, order.AddCommentReaction(database.Conn(), commentEvent.ID, otherUserID, CommentReactionThumbsUp))

		summaries, err := ListCommentReactionSummaries(database.Conn(), order.ID, []uuid.UUID{commentEvent.ID}, otherUserID)
		require.NoError(t, err)
		require.Len(t, summaries[commentEvent.ID], 1)
		assert.Equal(t, 2, summaries[commentEvent.ID][0].Count)
		assert.True(t, summaries[commentEvent.ID][0].ReactedByMe)

		summaries, err = ListCommentReactionSummaries(database.Conn(), order.ID, []uuid.UUID{commentEvent.ID}, uuid.New())
		require.NoError(t, err)
		require.Len(t, summaries[commentEvent.ID], 1)
		assert.Equal(t, 2, summaries[commentEvent.ID][0].Count)
		assert.False(t, summaries[commentEvent.ID][0].ReactedByMe)
	})

	t.Run("a different emoji on the same comment is a separate summary row", func(t *testing.T) {
		require.NoError(t, order.AddCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionRocket))

		summaries, err := ListCommentReactionSummaries(database.Conn(), order.ID, []uuid.UUID{commentEvent.ID}, userID)
		require.NoError(t, err)
		assert.Len(t, summaries[commentEvent.ID], 2)
	})

	t.Run("rejects reacting to an event that isn't a comment", func(t *testing.T) {
		require.NoError(t, order.RecordStatusUpdated(database.Conn(), statusUpdatedRecord{
			FromState: FactoryWorkOrderStateDraft,
			ToState:   FactoryWorkOrderStateOpen,
		}))

		events, err := order.ListEvents(database.Conn(), 10, nil)
		require.NoError(t, err)

		var statusEventID uuid.UUID
		for _, e := range events {
			if e.Type == factory.EventTypeOrderStatusUpdated {
				statusEventID = e.ID
				break
			}
		}
		require.NotEqual(t, uuid.Nil, statusEventID)

		err = order.AddCommentReaction(database.Conn(), statusEventID, userID, CommentReactionEyes)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderCommentNotFound)
	})

	t.Run("rejects reacting to a comment from another work order", func(t *testing.T) {
		otherOrder, err := factoryModel.CreateWorkOrder(database.Conn(), "Other order", "", &userID, nil, nil)
		require.NoError(t, err)

		err = otherOrder.AddCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionEyes)
		assert.ErrorIs(t, err, ErrFactoryWorkOrderCommentNotFound)
	})
}

func TestFactoryWorkOrder_RemoveCommentReaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, userID, factoryModel := setupFactoryWithUser(t, "reaction-remove")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Reaction removal target", "", &userID, nil, nil)
	require.NoError(t, err)

	userIDStr := userID.String()
	commentEvent, err := order.RecordCommentAdded(database.Conn(), "Removable", factory.WorkOrderCommentAuthor{
		Kind:   factory.CommentAuthorKindUser,
		UserID: &userIDStr,
	}, nil)
	require.NoError(t, err)

	require.NoError(t, order.AddCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionHeart))

	t.Run("removes an existing reaction", func(t *testing.T) {
		require.NoError(t, order.RemoveCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionHeart))

		summaries, err := ListCommentReactionSummaries(database.Conn(), order.ID, []uuid.UUID{commentEvent.ID}, userID)
		require.NoError(t, err)
		assert.Empty(t, summaries[commentEvent.ID])
	})

	t.Run("removing a reaction that doesn't exist is a no-op", func(t *testing.T) {
		require.NoError(t, order.RemoveCommentReaction(database.Conn(), commentEvent.ID, userID, CommentReactionHeart))
	})
}

func TestListCommentReactionSummaries_EmptyInput(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	summaries, err := ListCommentReactionSummaries(database.Conn(), uuid.New(), nil, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

func createSecondUser(t *testing.T, org *Organization, prefix string) uuid.UUID {
	t.Helper()

	nonce := time.Now().UnixNano()
	account, err := CreateAccount(
		fmt.Sprintf("Factory User %s %d", prefix, nonce),
		fmt.Sprintf("factory-%s-%d@example.com", prefix, nonce),
	)
	require.NoError(t, err)

	user, err := CreateUser(org.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)

	return user.ID
}
