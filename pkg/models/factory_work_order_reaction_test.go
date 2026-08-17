package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestIsValidWorkOrderReactionContent(t *testing.T) {
	for _, content := range ValidWorkOrderReactionContents {
		assert.True(t, IsValidWorkOrderReactionContent(content))
	}

	assert.False(t, IsValidWorkOrderReactionContent(""))
	assert.False(t, IsValidWorkOrderReactionContent("smile"))
	assert.False(t, IsValidWorkOrderReactionContent("+2"))
}

func TestFactoryWorkOrder_AddReaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "add-reaction")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	added, err := order.AddReaction(database.Conn(), userID, ReactionThumbsUp)
	require.NoError(t, err)
	assert.True(t, added, "first reaction should be created")

	reactions, err := order.ListReactions(database.Conn())
	require.NoError(t, err)
	require.Len(t, reactions, 1)
	assert.Equal(t, ReactionThumbsUp, reactions[0].Content)
	assert.Equal(t, userID, reactions[0].UserID)

	// Re-adding the same (user, content) pair is idempotent.
	added, err = order.AddReaction(database.Conn(), userID, ReactionThumbsUp)
	require.NoError(t, err)
	assert.False(t, added, "duplicate reaction should be a no-op")

	reactions, err = order.ListReactions(database.Conn())
	require.NoError(t, err)
	assert.Len(t, reactions, 1, "no duplicate row should be created")

	// The same user can hold a second, distinct reaction on the same order.
	added, err = order.AddReaction(database.Conn(), userID, ReactionHeart)
	require.NoError(t, err)
	assert.True(t, added)

	reactions, err = order.ListReactions(database.Conn())
	require.NoError(t, err)
	assert.Len(t, reactions, 2)
}

func TestFactoryWorkOrder_AddReaction_MultipleUsers(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "add-reaction-multi")

	account, err := CreateAccount("Second User", "add-reaction-multi-second-user@example.com")
	require.NoError(t, err)
	secondUser, err := CreateUser(factoryModel.OrganizationID, account.ID, account.Email, account.Name)
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	_, err = order.AddReaction(database.Conn(), userID, ReactionThumbsUp)
	require.NoError(t, err)
	_, err = order.AddReaction(database.Conn(), secondUser.ID, ReactionThumbsUp)
	require.NoError(t, err)

	reactions, err := order.ListReactions(database.Conn())
	require.NoError(t, err)
	assert.Len(t, reactions, 2, "both users' reactions on the same emoji should count separately")
}

func TestFactoryWorkOrder_RemoveReaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "remove-reaction")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	// Removing a reaction that was never added is a no-op.
	removed, err := order.RemoveReaction(database.Conn(), userID, ReactionRocket)
	require.NoError(t, err)
	assert.False(t, removed)

	_, err = order.AddReaction(database.Conn(), userID, ReactionRocket)
	require.NoError(t, err)
	_, err = order.AddReaction(database.Conn(), userID, ReactionEyes)
	require.NoError(t, err)

	removed, err = order.RemoveReaction(database.Conn(), userID, ReactionRocket)
	require.NoError(t, err)
	assert.True(t, removed)

	reactions, err := order.ListReactions(database.Conn())
	require.NoError(t, err)
	require.Len(t, reactions, 1, "removing one reaction shouldn't affect the user's other reactions")
	assert.Equal(t, ReactionEyes, reactions[0].Content)

	// Removing again is now a no-op.
	removed, err = order.RemoveReaction(database.Conn(), userID, ReactionRocket)
	require.NoError(t, err)
	assert.False(t, removed)
}

func TestFactoryWorkOrder_ListReactions_Empty(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "list-reactions-empty")

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	reactions, err := order.ListReactions(database.Conn())
	require.NoError(t, err)
	assert.Empty(t, reactions)
}
